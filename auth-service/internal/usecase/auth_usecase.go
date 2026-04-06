package usecase

import (
	"context"
	"time"

	jwt "auth-service/pkg/jwt"

	"auth-service/internal/domain"
	"auth-service/internal/domain/entity"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// AuthUseCase реализует бизнес-логику аутентификации.
type AuthUseCase struct {
	repo      UserRepo
	tokenRepo TokenRepo
	signKey   string
}

// New создаёт новый AuthUseCase.
func NewAuth(r UserRepo, tokenRepo TokenRepo, signKey string) *AuthUseCase {
	return &AuthUseCase{
		repo:      r,
		tokenRepo: tokenRepo,
		signKey:   signKey,
	}
}

// Register создаёт нового пользователя.
func (uc *AuthUseCase) Register(ctx context.Context, email, password, role string) (entity.User, error) {
	if !entity.IsValidRole(role) {
		return entity.User{}, domain.ErrInvalidRole
	}

	_, err := uc.repo.GetByEmail(ctx, email)
	if err == nil {
		return entity.User{}, domain.ErrUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return entity.User{}, domain.ErrPasswordChange
	}

	user := entity.User{
		ID:         uuid.New().String(),
		Email:      email,
		Password:   string(hashedPassword),
		Role:       role,
		LastOnline: time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err = uc.repo.Store(ctx, user); err != nil {
		return entity.User{}, domain.ErrUserAlreadyExists
	}

	return user, nil
}

// Login аутентифицирует пользователя и возвращает (accessToken, refreshToken).
func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (string, string, error) {
	user, err := uc.repo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", domain.ErrInvalidCredentials
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", "", domain.ErrInvalidCredentials
	}

	accessToken, err := jwt.GenerateAccessToken(user.ID, user.Email, user.Role, uc.signKey)
	if err != nil {
		return "", "", domain.ErrInvalidToken
	}

	refreshToken, err := jwt.GenerateRefreshToken(user.ID, uc.signKey)
	if err != nil {
		return "", "", domain.ErrInvalidToken
	}

	if err = uc.tokenRepo.StoreRefreshToken(ctx, user.ID, refreshToken); err != nil {
		return "", "", domain.ErrTokenRotation
	}

	return accessToken, refreshToken, nil
}

// RefreshTokens обновляет пару токенов по refresh токену (ротация).
func (uc *AuthUseCase) RefreshTokens(ctx context.Context, refreshToken string) (string, string, error) {
	userID, err := uc.tokenRepo.GetUserIDByRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", "", domain.ErrInvalidToken
	}

	// Удаляем старый токен (ротация)
	if err = uc.tokenRepo.DeleteRefreshToken(ctx, refreshToken); err != nil {
		return "", "", domain.ErrTokenRotation
	}

	user, err := uc.repo.GetByID(ctx, userID)
	if err != nil {
		return "", "", domain.ErrUserNotFound
	}

	newAccessToken, err := jwt.GenerateAccessToken(user.ID, user.Email, user.Role, uc.signKey)
	if err != nil {
		return "", "", domain.ErrInvalidToken
	}

	newRefreshToken, err := jwt.GenerateRefreshToken(user.ID, uc.signKey)
	if err != nil {
		return "", "", domain.ErrInvalidToken
	}

	if err = uc.tokenRepo.StoreRefreshToken(ctx, userID, newRefreshToken); err != nil {
		return "", "", domain.ErrTokenRotation
	}

	return newAccessToken, newRefreshToken, nil
}

// Logout инвалидирует refresh токен.
func (uc *AuthUseCase) Logout(ctx context.Context, refreshToken string) error {
	if err := uc.tokenRepo.DeleteRefreshToken(ctx, refreshToken); err != nil {
		return domain.ErrTokenRotation
	}
	return nil
}

// ChangePassword меняет пароль и инвалидирует все refresh токены пользователя.
func (uc *AuthUseCase) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := uc.repo.GetByID(ctx, userID)
	if err != nil {
		return domain.ErrUserNotFound
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return domain.ErrInvalidCredentials
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return domain.ErrPasswordChange
	}

	if err = uc.repo.UpdatePassword(ctx, userID, string(hashedPassword)); err != nil {
		return domain.ErrPasswordChange
	}

	// Инвалидируем все refresh токены после смены пароля
	if err = uc.tokenRepo.DeleteAllRefreshTokens(ctx, userID); err != nil {
		return domain.ErrTokenRotation
	}

	return nil
}
