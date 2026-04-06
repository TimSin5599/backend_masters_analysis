package usecase

import (
	"context"

	"auth-service/internal/domain/entity"
)

type (
	// Auth — бизнес-логика аутентификации.
	Auth interface {
		Register(ctx context.Context, email, password, role string) (entity.User, error)
		// Login возвращает (accessToken, refreshToken, error).
		Login(ctx context.Context, email, password string) (string, string, error)
		// RefreshTokens обновляет пару токенов по refresh токену (ротация).
		RefreshTokens(ctx context.Context, refreshToken string) (string, string, error)
		// Logout инвалидирует refresh токен.
		Logout(ctx context.Context, refreshToken string) error
		ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error
	}

	User interface {
		CreateUser(ctx context.Context, user entity.User) (string, error)
		GetByID(ctx context.Context, id string) (entity.User, error)
		ListUsers(ctx context.Context) ([]entity.User, error)
		UpdateUser(ctx context.Context, user entity.User) error
		UpdateLastOnline(ctx context.Context, id string) error
		DeleteUser(ctx context.Context, id string) error
	}

	// UserRepo — хранилище пользователей (PostgreSQL).
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

	// TokenRepo — хранилище refresh токенов (Redis).
	TokenRepo interface {
		StoreRefreshToken(ctx context.Context, userID, token string) error
		GetUserIDByRefreshToken(ctx context.Context, token string) (string, error)
		DeleteRefreshToken(ctx context.Context, token string) error
		DeleteAllRefreshTokens(ctx context.Context, userID string) error
	}
)
