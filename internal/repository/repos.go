package repository

import (
	"context"
	"github.com/EgorBessonov/gRPC/internal/model"
)

// Repository interface represent repository behavior
type Repository interface {
	Save(context.Context, *model.Order) error
	Get(context.Context, string) (*model.Order, error)
	Update(context.Context, *model.Order) error
	Delete(context.Context, string) error
	SaveAuthUser(context.Context, *model.AuthUser) error
	GetAuthUser(context.Context, string) (*model.AuthUser, error)
	GetAuthUserByID(context.Context, string) (*model.AuthUser, error)
	UpdateAuthUser(ctx context.Context, email, refreshToken string) error
	CloseDBConnection() error
}
