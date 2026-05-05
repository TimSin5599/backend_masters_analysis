package usecase

import (
	"context"
	"time"

	"auth-service/internal/domain"
	"auth-service/internal/domain/entity"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserUseCase реализует бизнес-логику для управления пользователями.
type UserUseCase struct {
	repo      UserRepo
	tokenRepo TokenRepo
}

// New создаёт новый UserUseCase.
func NewUser(r UserRepo, tokenRepo TokenRepo) *UserUseCase {
	return &UserUseCase{
		repo:      r,
		tokenRepo: tokenRepo,
	}
}

// CreateUser создаёт пользователя администратором.
func (uc *UserUseCase) CreateUser(ctx context.Context, user entity.User) (string, error) {
	if !entity.IsValidRoles(user.Roles) {
		return "", domain.ErrInvalidRole
	}

	_, err := uc.repo.GetByEmail(ctx, user.Email)
	if err == nil {
		return "", domain.ErrUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", domain.ErrPasswordChange
	}

	user.ID = uuid.New().String()
	user.Password = string(hashedPassword)
	user.LastOnline = time.Now()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	if err = uc.repo.Store(ctx, user); err != nil {
		return "", domain.ErrUserAlreadyExists
	}

	return user.ID, nil
}

// GetByID возвращает пользователя по ID.
func (uc *UserUseCase) GetByID(ctx context.Context, id string) (entity.User, error) {
	user, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return entity.User{}, domain.ErrUserNotFound
	}
	return user, nil
}

// ListUsers возвращает всех пользователей.
func (uc *UserUseCase) ListUsers(ctx context.Context) ([]entity.User, error) {
	return uc.repo.List(ctx)
}

// UpdateLastOnline обновляет время последнего онлайна.
func (uc *UserUseCase) UpdateLastOnline(ctx context.Context, id string) error {
	return uc.repo.UpdateLastOnline(ctx, id)
}

// UpdateUser обновляет данные пользователя.
func (uc *UserUseCase) UpdateUser(ctx context.Context, user entity.User) error {
	return uc.repo.Update(ctx, user)
}

// DeleteUser удаляет пользователя и все его refresh токены.
func (uc *UserUseCase) DeleteUser(ctx context.Context, id string) error {
	if err := uc.tokenRepo.DeleteAllRefreshTokens(ctx, id); err != nil {
		return domain.ErrTokenRotation
	}
	return uc.repo.Delete(ctx, id)
}