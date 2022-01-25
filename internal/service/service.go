package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/EgorBessonov/gRPC/internal/cache"
	"github.com/EgorBessonov/gRPC/internal/model"
	"github.com/EgorBessonov/gRPC/internal/repository"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
)

// Service type
type Service struct {
	rps   repository.Repository
	cache *cache.OrderCache
}

// NewService method returns new Service instance
func NewService(_rps repository.Repository, cache *cache.OrderCache) *Service {
	return &Service{rps: _rps, cache: cache}
}

const (
	accessTokenExTime  = 15
	refreshTokenExTime = 720
)

// CustomClaims struct represent user information in tokens
type CustomClaims struct {
	email    string
	userName string
	jwt.StandardClaims
}

// Registration method hash user password and after that save user in repository
func (s Service) Registration(ctx context.Context, authUser *model.AuthUser) error {
	hPassword, err := hashPassword(authUser.Password)
	if err != nil {
		return err
	}
	authUser.Password = hPassword
	err = s.rps.SaveAuthUser(ctx, authUser)
	if err != nil {
		return fmt.Errorf("service: registration failed - %w", err)
	}
	return nil
}

// RefreshToken method checks refresh token for validity and if it's ok return new token pair
func (s Service) RefreshToken(ctx context.Context, refreshTokenString string) (string, string, error) {
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("SECRETKEY")), nil
	}
	refreshToken, err := jwt.Parse(refreshTokenString, keyFunc)
	if err != nil {
		return "", "", fmt.Errorf("service: can't parse refresh token - %w", err)
	}
	if !refreshToken.Valid {
		return "", "", fmt.Errorf("service: expired refresh token")
	}
	claims := refreshToken.Claims.(jwt.MapClaims)
	userUUID := claims["jti"]
	if userUUID == "" {
		return "", "", fmt.Errorf("service: error while parsing claims")
	}
	authUser, err := s.rps.GetAuthUserByID(ctx, userUUID.(string))
	if err != nil {
		return "", "", fmt.Errorf("service: token refresh failed - %w", err)
	}
	if refreshTokenString != authUser.RefreshToken {
		return "", "", fmt.Errorf("service: invalid refresh token")
	}
	return createTokenPair(s.rps, ctx, authUser)
}

// Authentication method check user password for validity and if it's correct return access and refresh tokens
func (s Service) Authentication(ctx context.Context, email, password string) (string, string, error) {
	hashPassword, err := hashPassword(password)
	if err != nil {
		return "", "", err
	}
	authForm, err := s.rps.GetAuthUser(ctx, email)
	if err != nil {
		return "", "", fmt.Errorf("service: authentication failed - %w", err)
	}
	if authForm.Password != hashPassword {
		return "", "", fmt.Errorf("service: invalid password")
	}
	return createTokenPair(s.rps, ctx, authForm)
}

// UpdateAuthUser method update user instance in repository
func (s Service) UpdateAuthUser(ctx context.Context, email string, refreshToken string) error {
	return s.rps.UpdateAuthUser(ctx, email, refreshToken)
}

func ValidateToken(ctx context.Context) (bool, error) {
	tokenString, err := getTokenFormContext(ctx)
	if err != nil {
		return false, err
	}
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("SECRETKEY")), nil
	})
	if err != nil {
		return false, err
	}
	if !token.Valid {
		return false, fmt.Errorf("invalid or expired token.")
	}
	return true, nil
}

func createTokenPair(rps repository.Repository, ctx context.Context, authUser *model.AuthUser) (string, string, error) {
	expirationTimeAT := time.Now().Add(accessTokenExTime * time.Minute)
	expirationTimeRT := time.Now().Add(time.Hour * refreshTokenExTime)

	atClaims := &CustomClaims{
		userName: authUser.UserName,
		email:    authUser.Email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTimeAT.Unix(),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	accessTokenString, err := accessToken.SignedString([]byte(os.Getenv("SECRETKEY")))
	if err != nil {
		return "", "", fmt.Errorf("service: can't generate access token - %w", err)
	}

	rtClaims := &CustomClaims{
		userName: authUser.UserName,
		email:    authUser.Email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTimeRT.Unix(),
			Id:        authUser.UserUUID,
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(os.Getenv("SECRETKEY")))
	if err != nil {
		return "", "", fmt.Errorf("service: can't generate refresh token - %w", err)
	}

	err = rps.UpdateAuthUser(ctx, authUser.Email, refreshTokenString)
	if err != nil {
		return "", "", fmt.Errorf("service: can't set refresh token - %w", err)
	}
	return accessTokenString, refreshTokenString, nil
}

func hashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("service: zero password value")
	}
	h := sha256.New()
	h.Write([]byte(password))
	hashedPassword := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return hashedPassword, nil
}

func getTokenFormContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("service: can't get metadata from context")
	}
	accessToken, ok := md["accessToken"]
	if !ok || len(accessToken) == 0 {
		return "", fmt.Errorf("service: no token in metadata")
	}
	return accessToken[0], nil
}

// Save function method generate order uuid and after that save instance and repository
func (s Service) Save(ctx context.Context, order *model.Order) (string, error) {
	order.OrderID = uuid.New().String()
	err := s.cache.Save(order)
	if err != nil {
		return "", fmt.Errorf("service: can't create order - %w", err)
	}
	err = s.rps.Save(ctx, order)
	if err != nil {
		return "", fmt.Errorf("service: can't create order - %w", err)
	}
	return order.OrderID, nil
}

// Get method look through cache for order and if order wasn't found, method get it from repository and add it in cache
func (s Service) Get(ctx context.Context, orderID string) (*model.Order, error) {
	order, found := s.cache.Get(orderID) // add second param as ok
	if !found {
		order, err := s.rps.Get(ctx, orderID)
		if err != nil {
			return nil, fmt.Errorf("service: can't get order - %w", err)
		}
		err = s.cache.Save(order)
		if err != nil {
			return nil, fmt.Errorf("service: can't get order - %w", err)
		}
		return order, nil
	}
	return order, nil
}

// Delete method delete order from repository and cache
func (s Service) Delete(ctx context.Context, orderID string) error {
	err := s.cache.Delete(orderID)
	if err != nil {
		return fmt.Errorf("service: can't delete order - %e", err)
	}
	err = s.rps.Delete(ctx, orderID)
	if err != nil {
		return fmt.Errorf("service: can't delete order - %e", err)
	}
	return nil
}

// Update method update order instance in repository and cache
func (s Service) Update(ctx context.Context, order *model.Order) error {
	err := s.cache.Update(order)
	if err != nil {
		return fmt.Errorf("service: can't update order - %e", err)
	}
	err = s.rps.Update(ctx, order)
	if err != nil {
		return fmt.Errorf("service: can't update order - %e", err)
	}
	return nil
}

func (s Service) UploadImage(imageName string, imageData bytes.Buffer) error {
	imagePath := fmt.Sprintf("images/" + imageName)
	file, err := os.Create(imagePath)
	if err != nil {
		return fmt.Errorf("service: can't upload image - %e", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Errorf("error while closing os file instance - %e", err)
		}
	}()
	_, err = imageData.WriteTo(file)
	if err != nil {
		return fmt.Errorf("sevice: can't upload image - %e", err)
	}
	return nil
}
