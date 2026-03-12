package usecase

import (
	"auth-service/internal/entity"
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// AuthUseCase -
type AuthUseCase struct {
	repo       UserRepo
	signKey    []byte
	tokenTTL   time.Duration
}

// New -
func New(r UserRepo, signKey string, tokenTTL time.Duration) *AuthUseCase {
	return &AuthUseCase{
		repo:       r,
		signKey:    []byte(signKey),
		tokenTTL:   tokenTTL,
	}
}

// Register -
func (uc *AuthUseCase) Register(ctx context.Context, email, password, role string) (entity.User, error) {
	fmt.Printf("[USECASE] Регистрация - начало процесса для email: %s\n", email)

	// 0. Validate role
	if !entity.IsValidRole(role) {
		fmt.Printf("[USECASE] ❌ Невалидная роль: %s\n", role)
		return entity.User{}, fmt.Errorf("invalid role: %s", role)
	}

	// 1. Check if user exists
	fmt.Printf("[USECASE] Шаг 1: Проверка существования пользователя...\n")
	_, err := uc.repo.GetByEmail(ctx, email)
	if err == nil {
		fmt.Printf("[USECASE] ❌ Пользователь с email %s уже существует\n", email)
		return entity.User{}, fmt.Errorf("user already exists")
	}
	fmt.Printf("[USECASE] ✓ Пользователь не найден, продолжаем регистрацию\n")

	// 2. Hash password
	fmt.Printf("[USECASE] Шаг 2: Хеширование пароля...\n")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("[USECASE] ❌ Ошибка хеширования пароля: %v\n", err)
		return entity.User{}, fmt.Errorf("AuthUseCase - Register - bcrypt.GenerateFromPassword: %w", err)
	}
	fmt.Printf("[USECASE] ✓ Пароль успешно захеширован\n")

	userID := uuid.New().String()
	fmt.Printf("[USECASE] Шаг 3: Создание пользователя с ID: %s\n", userID)

	user := entity.User{
		ID:        userID,
		Email:     email,
		Password:  string(hashedPassword),
		Role:      role,
		LastOnline: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 3. Store user
	fmt.Printf("[USECASE] Шаг 4: Сохранение пользователя в базу данных...\n")
	err = uc.repo.Store(ctx, user)
	if err != nil {
		fmt.Printf("[USECASE] ❌ Ошибка сохранения в БД: %v\n", err)
		return entity.User{}, fmt.Errorf("AuthUseCase - Register - uc.repo.Store: %w", err)
	}
	fmt.Printf("[USECASE] ✅ Пользователь успешно сохранен в БД\n")

	return user, nil
}

// CreateUser -
func (uc *AuthUseCase) CreateUser(ctx context.Context, user entity.User) (string, error) {
	fmt.Printf("[USECASE] CreateUser - начало процесса для email: %s\n", user.Email)

	if !entity.IsValidRole(user.Role) {
		return "", fmt.Errorf("invalid role: %s", user.Role)
	}

	_, err := uc.repo.GetByEmail(ctx, user.Email)
	if err == nil {
		return "", fmt.Errorf("user with email %s already exists", user.Email)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	user.ID = uuid.New().String()
	user.Password = string(hashedPassword)
	user.LastOnline = time.Now()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	err = uc.repo.Store(ctx, user)
	if err != nil {
		return "", fmt.Errorf("failed to store user: %w", err)
	}

	fmt.Printf("[USECASE] ✅ Пользователь успешно создан | ID: %s | Email: %s\n", user.ID, user.Email)
	return user.ID, nil
}

// Login -
func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (string, error) {
	fmt.Printf("[USECASE] Авторизация - начало процесса для email: %s\n", email)

	fmt.Printf("[USECASE] Шаг 1: Поиск пользователя в БД...\n")
	user, err := uc.repo.GetByEmail(ctx, email)
	if err != nil {
		fmt.Printf("[USECASE] ❌ Пользователь не найден: %v\n", err)
		return "", fmt.Errorf("AuthUseCase - Login - uc.repo.GetByEmail: %w", err)
	}
	fmt.Printf("[USECASE] ✓ Пользователь найден | ID: %s | Роль: %s\n", user.ID, user.Role)

	// Check password
	fmt.Printf("[USECASE] Шаг 2: Проверка пароля...\n")
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		fmt.Printf("[USECASE] ❌ Неверный пароль\n")
		return "", fmt.Errorf("invalid credentials")
	}
	fmt.Printf("[USECASE] ✓ Пароль верный\n")

	// Generate Token
	fmt.Printf("[USECASE] Шаг 3: Генерация JWT токена...\n")
	// Calculate expiration time for logging purposes
	expirationTimeForLog := time.Now().Add(uc.tokenTTL)
	fmt.Printf("[USECASE] Токен будет действителен до: %s\n", expirationTimeForLog.Format("2006-01-02 15:04:05"))

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(uc.tokenTTL).Unix(),
		"user": map[string]interface{}{
			"email": user.Email,
			"role":  user.Role,
		},
	})

	tokenString, err := token.SignedString(uc.signKey)
	if err != nil {
		fmt.Printf("[USECASE] ❌ Ошибка подписи токена: %v\n", err)
		return "", fmt.Errorf("AuthUseCase - Login - token.SignedString: %w", err)
	}

	fmt.Printf("[USECASE] ✅ JWT токен успешно создан | UserID: %s\n", user.ID)
	return tokenString, nil
}

// GetByID -
func (uc *AuthUseCase) GetByID(ctx context.Context, id string) (entity.User, error) {
	fmt.Printf("[USECASE] GetByID - получение пользователя по ID: %s\n", id)

	user, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		fmt.Printf("[USECASE] ❌ Ошибка получения пользователя: %v\n", err)
		return entity.User{}, fmt.Errorf("AuthUseCase - GetByID - uc.repo.GetByID: %w", err)
	}

	fmt.Printf("[USECASE] ✅ Пользователь получен | Email: %s | Role: %s\n", user.Email, user.Role)
	return user, nil
}

// ListUsers -
func (uc *AuthUseCase) ListUsers(ctx context.Context) ([]entity.User, error) {
	fmt.Printf("[USECASE] ListUsers - получение списка пользователей\n")
	return uc.repo.List(ctx)
}

// UpdateLastOnline -
func (uc *AuthUseCase) UpdateLastOnline(ctx context.Context, id string) error {
	return uc.repo.UpdateLastOnline(ctx, id)
}

func (uc *AuthUseCase) UpdateUser(ctx context.Context, user entity.User) error {
	return uc.repo.Update(ctx, user)
}

func (uc *AuthUseCase) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	fmt.Printf("[USECASE] ChangePassword - начало процесса для userID: %s\n", userID)

	// 1. Get user
	user, err := uc.repo.GetByID(ctx, userID)
	if err != nil {
		fmt.Printf("[USECASE] ❌ Пользователь не найден: %v\n", err)
		return fmt.Errorf("user not found")
	}

	// 2. Check old password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword))
	if err != nil {
		fmt.Printf("[USECASE] ❌ Неверный старый пароль\n")
		return fmt.Errorf("invalid old password")
	}

	// 3. Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("[USECASE] ❌ Ошибка хеширования нового пароля: %v\n", err)
		return fmt.Errorf("internal server error")
	}

	// 4. Update password
	err = uc.repo.UpdatePassword(ctx, userID, string(hashedPassword))
	if err != nil {
		fmt.Printf("[USECASE] ❌ Ошибка обновления пароля в БД: %v\n", err)
		return fmt.Errorf("failed to update password")
	}

	fmt.Printf("[USECASE] ✅ Пароль успешно изменен для userID: %s\n", userID)
	return nil
}

func (uc *AuthUseCase) DeleteUser(ctx context.Context, id string) error {
	fmt.Printf("[USECASE] DeleteUser - удаление пользователя по ID: %s\n", id)
	return uc.repo.Delete(ctx, id)
}
