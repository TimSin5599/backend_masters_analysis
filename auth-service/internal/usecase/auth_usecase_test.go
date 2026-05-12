package usecase_test

import (
	"context"
	"testing"

	"auth-service/internal/domain"
	"auth-service/internal/domain/entity"
	"auth-service/internal/usecase"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

// ─── Mocks ───────────────────────────────────────────────────────────────────

type mockUserRepo struct {
	StoreFunc            func(ctx context.Context, user entity.User) error
	GetByEmailFunc       func(ctx context.Context, email string) (entity.User, error)
	GetByIDFunc          func(ctx context.Context, id string) (entity.User, error)
	ListFunc             func(ctx context.Context) ([]entity.User, error)
	UpdateFunc           func(ctx context.Context, user entity.User) error
	UpdateLastOnlineFunc func(ctx context.Context, id string) error
	UpdatePasswordFunc   func(ctx context.Context, id string, hashedPassword string) error
	DeleteFunc           func(ctx context.Context, id string) error
}

func (m *mockUserRepo) Store(ctx context.Context, user entity.User) error {
	if m.StoreFunc != nil {
		return m.StoreFunc(ctx, user)
	}
	return nil
}
func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (entity.User, error) {
	if m.GetByEmailFunc != nil {
		return m.GetByEmailFunc(ctx, email)
	}
	return entity.User{}, domain.ErrUserNotFound
}
func (m *mockUserRepo) GetByID(ctx context.Context, id string) (entity.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return entity.User{}, domain.ErrUserNotFound
}
func (m *mockUserRepo) List(ctx context.Context) ([]entity.User, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return nil, nil
}
func (m *mockUserRepo) Update(ctx context.Context, user entity.User) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, user)
	}
	return nil
}
func (m *mockUserRepo) UpdateLastOnline(ctx context.Context, id string) error {
	if m.UpdateLastOnlineFunc != nil {
		return m.UpdateLastOnlineFunc(ctx, id)
	}
	return nil
}
func (m *mockUserRepo) UpdatePassword(ctx context.Context, id string, hashedPassword string) error {
	if m.UpdatePasswordFunc != nil {
		return m.UpdatePasswordFunc(ctx, id, hashedPassword)
	}
	return nil
}
func (m *mockUserRepo) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

type mockTokenRepo struct {
	StoreRefreshTokenFunc       func(ctx context.Context, userID, token string) error
	GetUserIDByRefreshTokenFunc func(ctx context.Context, token string) (string, error)
	DeleteRefreshTokenFunc      func(ctx context.Context, token string) error
	DeleteAllRefreshTokensFunc  func(ctx context.Context, userID string) error
}

func (m *mockTokenRepo) StoreRefreshToken(ctx context.Context, userID, token string) error {
	if m.StoreRefreshTokenFunc != nil {
		return m.StoreRefreshTokenFunc(ctx, userID, token)
	}
	return nil
}
func (m *mockTokenRepo) GetUserIDByRefreshToken(ctx context.Context, token string) (string, error) {
	if m.GetUserIDByRefreshTokenFunc != nil {
		return m.GetUserIDByRefreshTokenFunc(ctx, token)
	}
	return "", domain.ErrInvalidToken
}
func (m *mockTokenRepo) DeleteRefreshToken(ctx context.Context, token string) error {
	if m.DeleteRefreshTokenFunc != nil {
		return m.DeleteRefreshTokenFunc(ctx, token)
	}
	return nil
}
func (m *mockTokenRepo) DeleteAllRefreshTokens(ctx context.Context, userID string) error {
	if m.DeleteAllRefreshTokensFunc != nil {
		return m.DeleteAllRefreshTokensFunc(ctx, userID)
	}
	return nil
}

// ─── Register ────────────────────────────────────────────────────────────────

func TestAuthUseCase_Register(t *testing.T) {
	t.Run("Invalid Role", func(t *testing.T) {
		uc := usecase.NewAuth(&mockUserRepo{}, &mockTokenRepo{}, "secret")
		_, err := uc.Register(context.Background(), "test@test.com", "password", "invalid_role")
		assert.ErrorIs(t, err, domain.ErrInvalidRole)
	})

	t.Run("User Already Exists", func(t *testing.T) {
		repo := &mockUserRepo{
			GetByEmailFunc: func(ctx context.Context, email string) (entity.User, error) {
				return entity.User{Email: email}, nil
			},
		}
		uc := usecase.NewAuth(repo, &mockTokenRepo{}, "secret")
		_, err := uc.Register(context.Background(), "test@test.com", "password", entity.RoleAdmin)
		assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
	})

	t.Run("Success", func(t *testing.T) {
		repo := &mockUserRepo{
			GetByEmailFunc: func(ctx context.Context, email string) (entity.User, error) {
				return entity.User{}, domain.ErrUserNotFound
			},
			StoreFunc: func(ctx context.Context, user entity.User) error { return nil },
		}
		uc := usecase.NewAuth(repo, &mockTokenRepo{}, "secret")
		user, err := uc.Register(context.Background(), "test@test.com", "password", entity.RoleAdmin)
		assert.NoError(t, err)
		assert.Equal(t, "test@test.com", user.Email)
		assert.Equal(t, []string{entity.RoleAdmin}, user.Roles)
	})
}

// ─── Login ───────────────────────────────────────────────────────────────────

func TestAuthUseCase_Login(t *testing.T) {
	hashed, _ := bcrypt.GenerateFromPassword([]byte("correctpass"), bcrypt.DefaultCost)

	t.Run("User Not Found Returns Invalid Credentials", func(t *testing.T) {
		repo := &mockUserRepo{
			GetByEmailFunc: func(ctx context.Context, email string) (entity.User, error) {
				return entity.User{}, domain.ErrUserNotFound
			},
		}
		uc := usecase.NewAuth(repo, &mockTokenRepo{}, "secret")
		_, _, err := uc.Login(context.Background(), "none@test.com", "pass")
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("Wrong Password", func(t *testing.T) {
		repo := &mockUserRepo{
			GetByEmailFunc: func(ctx context.Context, email string) (entity.User, error) {
				return entity.User{Email: email, Password: string(hashed), Roles: []string{entity.RoleAdmin}}, nil
			},
		}
		uc := usecase.NewAuth(repo, &mockTokenRepo{}, "secret")
		_, _, err := uc.Login(context.Background(), "test@test.com", "wrongpass")
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("Token Store Failure", func(t *testing.T) {
		repo := &mockUserRepo{
			GetByEmailFunc: func(ctx context.Context, email string) (entity.User, error) {
				return entity.User{ID: "uid1", Email: email, Password: string(hashed), Roles: []string{entity.RoleAdmin}}, nil
			},
		}
		tokenRepo := &mockTokenRepo{
			StoreRefreshTokenFunc: func(ctx context.Context, userID, token string) error {
				return domain.ErrTokenRotation
			},
		}
		uc := usecase.NewAuth(repo, tokenRepo, "secret")
		_, _, err := uc.Login(context.Background(), "test@test.com", "correctpass")
		assert.ErrorIs(t, err, domain.ErrTokenRotation)
	})

	t.Run("Success", func(t *testing.T) {
		repo := &mockUserRepo{
			GetByEmailFunc: func(ctx context.Context, email string) (entity.User, error) {
				return entity.User{ID: "uid1", Email: email, Password: string(hashed), Roles: []string{entity.RoleAdmin}}, nil
			},
		}
		uc := usecase.NewAuth(repo, &mockTokenRepo{}, "secret")
		access, refresh, err := uc.Login(context.Background(), "test@test.com", "correctpass")
		assert.NoError(t, err)
		assert.NotEmpty(t, access)
		assert.NotEmpty(t, refresh)
	})
}

// ─── RefreshTokens ───────────────────────────────────────────────────────────

func TestAuthUseCase_RefreshTokens(t *testing.T) {
	t.Run("Invalid Refresh Token", func(t *testing.T) {
		uc := usecase.NewAuth(&mockUserRepo{}, &mockTokenRepo{}, "secret")
		_, _, err := uc.RefreshTokens(context.Background(), "bad-token")
		assert.ErrorIs(t, err, domain.ErrInvalidToken)
	})

	t.Run("User Not Found After Token Lookup", func(t *testing.T) {
		tokenRepo := &mockTokenRepo{
			GetUserIDByRefreshTokenFunc: func(ctx context.Context, token string) (string, error) {
				return "uid1", nil
			},
		}
		uc := usecase.NewAuth(&mockUserRepo{}, tokenRepo, "secret")
		_, _, err := uc.RefreshTokens(context.Background(), "valid-token")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("Success", func(t *testing.T) {
		repo := &mockUserRepo{
			GetByIDFunc: func(ctx context.Context, id string) (entity.User, error) {
				return entity.User{ID: id, Email: "u@t.com", Roles: []string{entity.RoleAdmin}}, nil
			},
		}
		tokenRepo := &mockTokenRepo{
			GetUserIDByRefreshTokenFunc: func(ctx context.Context, token string) (string, error) {
				return "uid1", nil
			},
		}
		uc := usecase.NewAuth(repo, tokenRepo, "secret")
		access, refresh, err := uc.RefreshTokens(context.Background(), "valid-token")
		assert.NoError(t, err)
		assert.NotEmpty(t, access)
		assert.NotEmpty(t, refresh)
	})
}

// ─── Logout ──────────────────────────────────────────────────────────────────

func TestAuthUseCase_Logout(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		uc := usecase.NewAuth(&mockUserRepo{}, &mockTokenRepo{}, "secret")
		err := uc.Logout(context.Background(), "some-token")
		assert.NoError(t, err)
	})

	t.Run("Delete Failure Returns TokenRotation", func(t *testing.T) {
		tokenRepo := &mockTokenRepo{
			DeleteRefreshTokenFunc: func(ctx context.Context, token string) error {
				return domain.ErrTokenRotation
			},
		}
		uc := usecase.NewAuth(&mockUserRepo{}, tokenRepo, "secret")
		err := uc.Logout(context.Background(), "some-token")
		assert.ErrorIs(t, err, domain.ErrTokenRotation)
	})
}

// ─── ChangePassword ──────────────────────────────────────────────────────────

func TestAuthUseCase_ChangePassword(t *testing.T) {
	hashed, _ := bcrypt.GenerateFromPassword([]byte("oldpass"), bcrypt.DefaultCost)

	t.Run("User Not Found", func(t *testing.T) {
		uc := usecase.NewAuth(&mockUserRepo{}, &mockTokenRepo{}, "secret")
		err := uc.ChangePassword(context.Background(), "uid1", "oldpass", "newpass")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("Wrong Old Password", func(t *testing.T) {
		repo := &mockUserRepo{
			GetByIDFunc: func(ctx context.Context, id string) (entity.User, error) {
				return entity.User{ID: id, Password: string(hashed)}, nil
			},
		}
		uc := usecase.NewAuth(repo, &mockTokenRepo{}, "secret")
		err := uc.ChangePassword(context.Background(), "uid1", "wrongpass", "newpass")
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("Success", func(t *testing.T) {
		repo := &mockUserRepo{
			GetByIDFunc: func(ctx context.Context, id string) (entity.User, error) {
				return entity.User{ID: id, Password: string(hashed)}, nil
			},
		}
		uc := usecase.NewAuth(repo, &mockTokenRepo{}, "secret")
		err := uc.ChangePassword(context.Background(), "uid1", "oldpass", "newpass")
		assert.NoError(t, err)
	})
}
