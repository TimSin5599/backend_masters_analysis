package repository

import (
	"context"
	"fmt"

	"auth-service/internal/domain/entity"
	"auth-service/internal/usecase"
	"auth-service/pkg/postgres"

	"github.com/jackc/pgx/v5"
)

type PostgresRepo struct {
	*postgres.Postgres
}

func NewPGRepo(pg *postgres.Postgres) *PostgresRepo {
	return &PostgresRepo{pg}
}

// Verify adherence to interface
var _ usecase.UserRepo = (*PostgresRepo)(nil)

func (r *PostgresRepo) Store(ctx context.Context, user entity.User) error {
	fmt.Printf("[REPOSITORY] Store - Сохранение пользователя в БД\n")
	fmt.Printf("[REPOSITORY] ID: %s | Email: %s | Roles: %v\n", user.ID, user.Email, user.Roles)

	query := `INSERT INTO users (id, email, password, roles, first_name, last_name, phone, avatar_path, last_online, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.Pool.Exec(ctx, query, user.ID, user.Email, user.Password, user.Roles, user.FirstName, user.LastName, user.Phone, user.AvatarPath, user.LastOnline, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		fmt.Printf("[REPOSITORY] ❌ Ошибка выполнения SQL запроса: %v\n", err)
		return fmt.Errorf("PostgresRepo - Store - r.Pool.Exec: %w", err)
	}

	fmt.Printf("[REPOSITORY] ✅ Пользователь успешно сохранен в БД\n")
	return nil
}

func (r *PostgresRepo) GetByEmail(ctx context.Context, email string) (entity.User, error) {
	fmt.Printf("[REPOSITORY] GetByEmail - Поиск пользователя по email: %s\n", email)

	query := `SELECT id, email, password, roles, COALESCE(first_name, ''), COALESCE(last_name, ''), COALESCE(phone, ''), COALESCE(avatar_path, ''), COALESCE(last_online, '1970-01-01 00:00:00'), created_at, updated_at FROM users WHERE email = $1`

	var user entity.User
	err := r.Pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Roles,
		&user.FirstName,
		&user.LastName,
		&user.Phone,
		&user.AvatarPath,
		&user.LastOnline,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			fmt.Printf("[REPOSITORY] ❌ Пользователь с email %s не найден\n", email)
			return entity.User{}, fmt.Errorf("user not found")
		}
		fmt.Printf("[REPOSITORY] ❌ Ошибка выполнения SQL запроса: %v\n", err)
		return entity.User{}, fmt.Errorf("PostgresRepo - GetByEmail - r.Pool.QueryRow.Scan: %w", err)
	}

	fmt.Printf("[REPOSITORY] ✅ Пользователь найден | ID: %s | Roles: %v\n", user.ID, user.Roles)
	return user, nil
}

func (r *PostgresRepo) GetByID(ctx context.Context, id string) (entity.User, error) {
	fmt.Printf("[REPOSITORY] GetByID - Поиск пользователя по ID: %s\n", id)

	query := `SELECT id, email, password, roles, COALESCE(first_name, ''), COALESCE(last_name, ''), COALESCE(phone, ''), COALESCE(avatar_path, ''), COALESCE(last_online, '1970-01-01 00:00:00'), created_at, updated_at FROM users WHERE id = $1`

	var user entity.User
	err := r.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Roles,
		&user.FirstName,
		&user.LastName,
		&user.Phone,
		&user.AvatarPath,
		&user.LastOnline,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			fmt.Printf("[REPOSITORY] ❌ Пользователь с ID %s не найден\n", id)
			return entity.User{}, fmt.Errorf("user not found")
		}
		fmt.Printf("[REPOSITORY] ❌ Ошибка выполнения SQL запроса: %v\n", err)
		return entity.User{}, fmt.Errorf("PostgresRepo - GetByID - r.Pool.QueryRow.Scan: %w", err)
	}

	fmt.Printf("[REPOSITORY] ✅ Пользователь найден | Email: %s | Roles: %v\n", user.Email, user.Roles)
	return user, nil
}

func (r *PostgresRepo) List(ctx context.Context) ([]entity.User, error) {
	query := `SELECT id, email, roles, COALESCE(first_name, ''), COALESCE(last_name, ''), COALESCE(phone, ''), COALESCE(avatar_path, ''), COALESCE(last_online, '1970-01-01 00:00:00'), created_at, updated_at FROM users`
	rows, err := r.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("PostgresRepo - List - r.Pool.Query: %w", err)
	}
	defer rows.Close()

	users := []entity.User{}
	for rows.Next() {
		var user entity.User
		err = rows.Scan(
			&user.ID,
			&user.Email,
			&user.Roles,
			&user.FirstName,
			&user.LastName,
			&user.Phone,
			&user.AvatarPath,
			&user.LastOnline,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("PostgresRepo - List - rows.Scan: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *PostgresRepo) Update(ctx context.Context, user entity.User) error {
	query := `UPDATE users SET first_name = $1, last_name = $2, phone = $3, avatar_path = $4, roles = $5, updated_at = NOW() WHERE id = $6`
	_, err := r.Pool.Exec(ctx, query, user.FirstName, user.LastName, user.Phone, user.AvatarPath, user.Roles, user.ID)
	if err != nil {
		return fmt.Errorf("PostgresRepo - Update - r.Pool.Exec: %w", err)
	}
	return nil
}

func (r *PostgresRepo) UpdatePassword(ctx context.Context, id string, hashedPassword string) error {
	query := `UPDATE users SET password = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.Pool.Exec(ctx, query, hashedPassword, id)
	if err != nil {
		return fmt.Errorf("PostgresRepo - UpdatePassword - r.Pool.Exec: %w", err)
	}
	return nil
}

func (r *PostgresRepo) UpdateLastOnline(ctx context.Context, id string) error {
	query := `UPDATE users SET last_online = NOW() WHERE id = $1`
	_, err := r.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("PostgresRepo - UpdateLastOnline - r.Pool.Exec: %w", err)
	}
	return nil
}

// Delete удаляет пользователя по ID
func (r *PostgresRepo) Delete(ctx context.Context, id string) error {
	fmt.Printf("[PG_REPO] Delete - удаление пользователя: %s\n", id)
	query := `DELETE FROM users WHERE id = $1`
	cmdTag, err := r.Pool.Exec(ctx, query, id)
	if err != nil {
		fmt.Printf("[PG_REPO] ❌ Ошибка выполнения запроса: %v\n", err)
		return fmt.Errorf("PostgresRepo - Delete - r.Pool.Exec: %w", err)
	}

	rowsAffected := cmdTag.RowsAffected()
	if rowsAffected == 0 {
		fmt.Printf("[PG_REPO] ❌ Пользователь с ID %s не найден\n", id)
		return fmt.Errorf("PostgresRepo - Delete: user not found")
	}

	fmt.Printf("[PG_REPO] ✅ Пользователь успешно удален. Затронуто строк: %d\n", rowsAffected)
	return nil
}
