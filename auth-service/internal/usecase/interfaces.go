package usecase

import (
	"auth-service/internal/entity"
	"context"
)

type (
	// Auth interface
	Auth interface {
		Register(ctx context.Context, email, password, role string) (entity.User, error)
		CreateUser(ctx context.Context, user entity.User) (string, error)
		Login(ctx context.Context, email, password string) (string, error) // Returns JWT token
		GetByID(ctx context.Context, id string) (entity.User, error)
		ListUsers(ctx context.Context) ([]entity.User, error)
		UpdateUser(ctx context.Context, user entity.User) error
		UpdateLastOnline(ctx context.Context, id string) error
		ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error
		DeleteUser(ctx context.Context, id string) error
	}

	// UserRepo interface
	UserRepo interface {
		Store(ctx context.Context, user entity.User) error
		GetByEmail(ctx context.Context, email string) (entity.User, error)
		GetByID(ctx context.Context, id string) (entity.User, error)
		List(ctx context.Context) ([]entity.User, error)
		Update(ctx context.Context, user entity.User) error
		UpdateLastOnline(ctx context.Context, id string) error
		UpdatePassword(ctx context.Context, id string, hashedPassword string) error
		Delete(ctx context.Context, id string) error
	}
)
