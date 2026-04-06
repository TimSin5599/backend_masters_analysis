package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"auth-service/internal/domain/entity"
	"auth-service/internal/usecase"
)

type UserRepo struct {
	mu    sync.RWMutex
	users map[string]entity.User
}

func New() *UserRepo {
	return &UserRepo{
		users: make(map[string]entity.User),
	}
}

var _ usecase.UserRepo = (*UserRepo)(nil)

func (r *UserRepo) Store(ctx context.Context, user entity.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.users[user.Email] = user
	return nil
}

// Delete удаляет пользователя из памяти (для тестов, если нужно)
func (r *UserRepo) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[id]; !ok {
		return fmt.Errorf("user not found")
	}

	delete(r.users, id)
	return nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (entity.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[email]
	if !ok {
		return entity.User{}, fmt.Errorf("user not found")
	}

	return user, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (entity.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Поскольку это in-memory репозиторий с map по email, нужно перебрать все записи
	for _, user := range r.users {
		if user.ID == id {
			return user, nil
		}
	}

	return entity.User{}, fmt.Errorf("user not found")
}

func (r *UserRepo) List(ctx context.Context) ([]entity.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]entity.User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}

	return users, nil
}

func (r *UserRepo) Update(ctx context.Context, user entity.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Since map is by email, we should check by ID
	for email, u := range r.users {
		if u.ID == user.ID {
			u.FirstName = user.FirstName
			u.LastName = user.LastName
			u.Phone = user.Phone
			u.AvatarPath = user.AvatarPath
			u.Role = user.Role
			u.UpdatedAt = time.Now()
			r.users[email] = u
			return nil
		}
	}

	return fmt.Errorf("user not found")
}

func (r *UserRepo) UpdateLastOnline(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for email, user := range r.users {
		if user.ID == id {
			user.LastOnline = time.Now()
			r.users[email] = user
			return nil
		}
	}

	return fmt.Errorf("user not found")
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id string, hashedPassword string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for email, user := range r.users {
		if user.ID == id {
			user.Password = hashedPassword
			user.UpdatedAt = time.Now()
			r.users[email] = user
			return nil
		}
	}

	return fmt.Errorf("user not found")
}
