package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/EgorBessonov/gRPC/internal/model"
	ordercrud "github.com/EgorBessonov/gRPC/internal/protocol"
	"github.com/EgorBessonov/gRPC/internal/service"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	s *service.Service
	ordercrud.UnimplementedCRUDServer
}

func NewServer(s *service.Service) *Server {
	return &Server{s: s}
}

// SaveOrder method for saving order
func (s Server) SaveOrder(ctx context.Context, request *ordercrud.SaveOrderRequest) (*ordercrud.SaveOrderResponse, error) {
	/*if ok, err := h.s.ValidateToken(ctx); !ok {
		log.Errorf("handler: token validation failed - %e", err)
		return nil, err
	}*/
	order := model.Order{
		OrderID:     request.Order.OrderId,
		OrderName:   request.Order.OrderName,
		OrderCost:   int(request.Order.OrderCost),
		IsDelivered: request.Order.IsDelivered,
	}
	orderID, err := s.s.Save(ctx, &order)
	if err != nil {
		log.Error(fmt.Errorf("handler: can't save order - %e", err))
		return nil, err
	}
	return &ordercrud.SaveOrderResponse{OrderId: orderID}, nil
}

// GetOrder method return order instance with selection by orderID
func (s Server) GetOrder(ctx context.Context, request *ordercrud.GetOrderRequest) (*ordercrud.GetOrderResponse, error) {
	/*if ok, err := h.s.ValidateToken(ctx); !ok {
		log.Errorf("handler: token validation failed - %e", err)
		return nil, err
	}*/
	order, err := s.s.Get(ctx, request.OrderId)
	if err != nil {
		log.Error(fmt.Errorf("handler: can't get order - %e", err))
		return nil, err
	}
	return &ordercrud.GetOrderResponse{
		Order: &ordercrud.Order{
			OrderId:     order.OrderID,
			OrderName:   order.OrderName,
			OrderCost:   int32(order.OrderCost),
			IsDelivered: order.IsDelivered,
		},
	}, nil
}

// DeleteOrder method delete order instance from repository
func (s Server) DeleteOrder(ctx context.Context, request *ordercrud.DeleteOrderRequest) (*ordercrud.DeleteOrderResponse, error) {
	/*if ok, err := h.s.ValidateToken(ctx); !ok {
		log.Errorf("handler: token validation failed - %e", err)
		return nil, err
	}*/
	err := s.s.Delete(ctx, request.OrderId)
	if err != nil {
		log.Error(fmt.Errorf("handler: can't get order - %e", err))
		return nil, err
	}
	return &ordercrud.DeleteOrderResponse{Result: fmt.Sprint("success")}, nil
}

// UpdateOrder method update order instance
func (s Server) UpdateOrder(ctx context.Context, request *ordercrud.UpdateOrderRequest) (*ordercrud.UpdateOrderResponse, error) {
	/*if ok, err := h.s.ValidateToken(ctx); !ok {
		log.Errorf("handler: token validation failed - %e", err)
		return nil, err
	}*/
	order := model.Order{
		OrderID:     request.Order.OrderId,
		OrderName:   request.Order.OrderName,
		OrderCost:   int(request.Order.OrderCost),
		IsDelivered: request.Order.IsDelivered,
	}
	err := s.s.Update(ctx, &order)
	if err != nil {
		log.Errorf("handler: can't update order - %e", err)
		return nil, err
	}
	return &ordercrud.UpdateOrderResponse{Result: fmt.Sprint("success")}, nil
}

// Authentication method checks user password and if it ok return access and refresh tokens
func (s Server) Authentication(ctx context.Context, request *ordercrud.AuthenticationRequest) (*ordercrud.AuthenticationResponse, error) {
	accessToken, refreshToken, err := s.s.Authentication(ctx, request.Email, request.Password)
	if err != nil {
		log.Errorf("handler: authentication failed - %e", err)
		return nil, err
	}
	return &ordercrud.AuthenticationResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken}, nil
}

// Registration - method for user creation
func (s Server) Registration(ctx context.Context, request *ordercrud.RegistrationRequest) (*ordercrud.RegistrationResponse, error) {
	authUser := model.AuthUser{
		UserUUID: request.AuthUser.UserUuid,
		UserName: request.AuthUser.UserName,
		Email:    request.AuthUser.Email,
		Password: request.AuthUser.Password,
	}
	err := s.s.Registration(ctx, &authUser)
	if err != nil {
		log.Errorf("handler: registration failed - %e", err)
		return nil, err
	}
	return &ordercrud.RegistrationResponse{Result: fmt.Sprint("success")}, nil
}

// RefreshToken - method checks refresh token for validity and if it's ok return new token pair
func (s Server) RefreshToken(ctx context.Context, request *ordercrud.RefreshTokenRequest) (*ordercrud.RefreshTokenResponse, error) {
	rToken := request.RefreshToken
	if rToken == "" {
		log.Error("handler: token refresh failed - empty value")
		return nil, errors.New("empty refreshToken value")
	}
	accessToken, refreshToken, err := s.s.RefreshToken(ctx, rToken)
	if err != nil {
		log.Errorf("handler: token refresh failed - %e", err)
		return nil, err
	}
	return &ordercrud.RefreshTokenResponse{RefreshToken: refreshToken, AccessToken: accessToken}, nil
}

// Logout method delete user refresh token from repository
func (s Server) Logout(ctx context.Context, request *ordercrud.LogoutRequest) (*ordercrud.LogoutResponse, error) {
	email := request.Email
	if email == "" {
		log.Error("handler: logout failed - empty value")
		return nil, errors.New("empty email value")
	}
	err := s.s.UpdateAuthUser(ctx, email, "")
	if err != nil {
		log.Errorf("handler: logout failed - %e", err)
		return nil, err
	}
	return &ordercrud.LogoutResponse{Result: fmt.Sprint("success")}, nil
}
