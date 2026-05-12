package usecase_test

import (
	"context"
	"testing"

	"auth-service/internal/domain"
	"auth-service/internal/domain/entity"
	"auth-service/internal/usecase"

	"github.com/stretchr/testify/assert"
)

func TestUserUseCase_CreateUser(t *testing.T) {
	t.Run("Invalid Role", func(t *testing.T) {
		uc := usecase.NewUser(&mockUserRepo{}, &mockTokenRepo{})
		_, err := uc.CreateUser(context.Background(), entity.User{Email: "a@b.com", Password: "pass", Roles: []string{"unknown"}})
		assert.ErrorIs(t, err, domain.ErrInvalidRole)
	})

	t.Run("User Already Exists", func(t *testing.T) {
		repo := &mockUserRepo{
			GetByEmailFunc: func(ctx context.Context, email string) (entity.User, error) {
				return entity.User{Email: email}, nil
			},
		}
		uc := usecase.NewUser(repo, &mockTokenRepo{})
		_, err := uc.CreateUser(context.Background(), entity.User{Email: "a@b.com", Password: "pass", Roles: []string{entity.RoleAdmin}})
		assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
	})

	t.Run("Success", func(t *testing.T) {
		repo := &mockUserRepo{
			GetByEmailFunc: func(ctx context.Context, email string) (entity.User, error) {
				return entity.User{}, domain.ErrUserNotFound
			},
			StoreFunc: func(ctx context.Context, user entity.User) error { return nil },
		}
		uc := usecase.NewUser(repo, &mockTokenRepo{})
		id, err := uc.CreateUser(context.Background(), entity.User{Email: "a@b.com", Password: "pass", Roles: []string{entity.RoleAdmin}})
		assert.NoError(t, err)
		assert.NotEmpty(t, id)
	})
}

func TestUserUseCase_GetByID(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		repo := &mockUserRepo{
			GetByIDFunc: func(ctx context.Context, id string) (entity.User, error) {
				return entity.User{ID: id, Email: "a@b.com"}, nil
			},
		}
		uc := usecase.NewUser(repo, &mockTokenRepo{})
		user, err := uc.GetByID(context.Background(), "uid1")
		assert.NoError(t, err)
		assert.Equal(t, "uid1", user.ID)
	})

	t.Run("Not Found", func(t *testing.T) {
		uc := usecase.NewUser(&mockUserRepo{}, &mockTokenRepo{})
		_, err := uc.GetByID(context.Background(), "missing")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

func TestUserUseCase_ListUsers(t *testing.T) {
	repo := &mockUserRepo{
		ListFunc: func(ctx context.Context) ([]entity.User, error) {
			return []entity.User{{ID: "1"}, {ID: "2"}}, nil
		},
	}
	uc := usecase.NewUser(repo, &mockTokenRepo{})
	users, err := uc.ListUsers(context.Background())
	assert.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestUserUseCase_DeleteUser(t *testing.T) {
	t.Run("Token Deletion Failure", func(t *testing.T) {
		tokenRepo := &mockTokenRepo{
			DeleteAllRefreshTokensFunc: func(ctx context.Context, userID string) error {
				return domain.ErrTokenRotation
			},
		}
		uc := usecase.NewUser(&mockUserRepo{}, tokenRepo)
		err := uc.DeleteUser(context.Background(), "uid1")
		assert.ErrorIs(t, err, domain.ErrTokenRotation)
	})

	t.Run("Success", func(t *testing.T) {
		uc := usecase.NewUser(&mockUserRepo{}, &mockTokenRepo{})
		err := uc.DeleteUser(context.Background(), "uid1")
		assert.NoError(t, err)
	})
}
