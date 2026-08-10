package service

import (
	"context"
	"errors"

	"user/internal/model/entity"
)

var ErrUserNotFound = errors.New("user not found")

// UserService contains user business operations required by RPC handlers.
type UserService interface {
	FindByUsername(ctx context.Context, username string) (*entity.User, error)
}
