package usecase_test

import (
	"context"
	"testing"

	"auth-service/internal/domain"
	"auth-service/internal/domain/entity"
	"auth-service/internal/usecase"
	"github.com/stretchr/testify/assert"
)

type mockUserRepo struct {
	StoreFunc      func(ctx context.Context, user entity.User) error
	GetByEmailFunc func(ctx context.Context, email string) (entity.User, error)
}

func (m *mockUserRepo) Store(ctx context.Context, user entity.User) error {
	return m.StoreFunc(ctx, user)
}
func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (entity.User, error) {
	return m.GetByEmailFunc(ctx, email)
}
func (m *mockUserRepo) GetByID(ctx context.Context, id string) (entity.User, error)           { return entity.User{}, nil }
func (m *mockUserRepo) List(ctx context.Context) ([]entity.User, error)                       { return nil, nil }
func (m *mockUserRepo) Update(ctx context.Context, user entity.User) error                    { return nil }
func (m *mockUserRepo) UpdateLastOnline(ctx context.Context, id string) error                 { return nil }
func (m *mockUserRepo) UpdatePassword(ctx context.Context, id string, hashedPassword string) error { return nil }
func (m *mockUserRepo) Delete(ctx context.Context, id string) error                           { return nil }

type mockTokenRepo struct{}

func (m *mockTokenRepo) StoreRefreshToken(ctx context.Context, userID, token string) error {
	return nil
}
func (m *mockTokenRepo) GetUserIDByRefreshToken(ctx context.Context, token string) (string, error) {
	return "", nil
}
func (m *mockTokenRepo) DeleteRefreshToken(ctx context.Context, token string) error { return nil }
func (m *mockTokenRepo) DeleteAllRefreshTokens(ctx context.Context, userID string) error {
	return nil
}

func TestAuthUseCase_Register(t *testing.T) {
	t.Run("Invalid Role", func(t *testing.T) {
		repo := &mockUserRepo{}
		tokenRepo := &mockTokenRepo{}
		uc := usecase.NewAuth(repo, tokenRepo, "secret")

		_, err := uc.Register(context.Background(), "test@test.com", "password", "invalid_role")
		assert.ErrorIs(t, err, domain.ErrInvalidRole)
	})

	t.Run("User Already Exists", func(t *testing.T) {
		repo := &mockUserRepo{
			GetByEmailFunc: func(ctx context.Context, email string) (entity.User, error) {
				return entity.User{Email: email}, nil // Found
			},
		}
		tokenRepo := &mockTokenRepo{}
		uc := usecase.NewAuth(repo, tokenRepo, "secret")

		_, err := uc.Register(context.Background(), "test@test.com", "password", entity.RoleAdmin)
		assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
	})

	t.Run("Success", func(t *testing.T) {
		repo := &mockUserRepo{
			GetByEmailFunc: func(ctx context.Context, email string) (entity.User, error) {
				return entity.User{}, domain.ErrUserNotFound // Not found
			},
			StoreFunc: func(ctx context.Context, user entity.User) error {
				return nil
			},
		}
		tokenRepo := &mockTokenRepo{}
		uc := usecase.NewAuth(repo, tokenRepo, "secret")

		user, err := uc.Register(context.Background(), "test@test.com", "password", entity.RoleAdmin)
		assert.NoError(t, err)
		assert.Equal(t, "test@test.com", user.Email)
		assert.Equal(t, entity.RoleAdmin, user.Role)
	})
}
