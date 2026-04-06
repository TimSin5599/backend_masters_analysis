package repository

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	refreshTokenPrefix = "refresh:"
	userTokensPrefix   = "user_tokens:"
	refreshTTL         = 7 * 24 * time.Hour
)

// RedisRepo — хранилище refresh токенов в Redis.
type RedisRepo struct {
	client *redis.Client
}

// NewRedisRepo создаёт новый RedisRepo.
func NewRedisRepo(client *redis.Client) *RedisRepo {
	return &RedisRepo{client: client}
}

// hashToken возвращает SHA-256 хеш токена в hex-формате.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}

// StoreRefreshToken сохраняет refresh токен в Redis с TTL 7 дней.
func (r *RedisRepo) StoreRefreshToken(ctx context.Context, userID, token string) error {
	hash := hashToken(token)
	pipe := r.client.Pipeline()

	// SET refresh:{hash} userID EX 604800
	pipe.Set(ctx, refreshTokenPrefix+hash, userID, refreshTTL)
	// SADD user_tokens:{userID} hash (для инвалидации всех токенов пользователя)
	pipe.SAdd(ctx, userTokensPrefix+userID, hash)
	// Обновляем TTL у сета пользователя, чтобы не копились "мёртвые" ключи
	pipe.Expire(ctx, userTokensPrefix+userID, refreshTTL)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("RedisRepo - StoreRefreshToken - pipe.Exec: %w", err)
	}
	return nil
}

// GetUserIDByRefreshToken возвращает userID по refresh токену.
// Возвращает пустую строку и ошибку, если токен не найден / истёк.
func (r *RedisRepo) GetUserIDByRefreshToken(ctx context.Context, token string) (string, error) {
	hash := hashToken(token)
	userID, err := r.client.Get(ctx, refreshTokenPrefix+hash).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("RedisRepo - GetUserIDByRefreshToken: token not found or expired")
	}
	if err != nil {
		return "", fmt.Errorf("RedisRepo - GetUserIDByRefreshToken - client.Get: %w", err)
	}
	return userID, nil
}

// DeleteRefreshToken удаляет один refresh токен (logout / ротация).
func (r *RedisRepo) DeleteRefreshToken(ctx context.Context, token string) error {
	hash := hashToken(token)

	// Получаем userID чтобы убрать из сета
	userID, err := r.client.Get(ctx, refreshTokenPrefix+hash).Result()
	if err == redis.Nil {
		return nil // уже удалён — не ошибка
	}
	if err != nil {
		return fmt.Errorf("RedisRepo - DeleteRefreshToken - client.Get: %w", err)
	}

	pipe := r.client.Pipeline()
	pipe.Del(ctx, refreshTokenPrefix+hash)
	pipe.SRem(ctx, userTokensPrefix+userID, hash)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("RedisRepo - DeleteRefreshToken - pipe.Exec: %w", err)
	}
	return nil
}

// DeleteAllRefreshTokens инвалидирует все refresh токены пользователя (при смене пароля).
func (r *RedisRepo) DeleteAllRefreshTokens(ctx context.Context, userID string) error {
	hashes, err := r.client.SMembers(ctx, userTokensPrefix+userID).Result()
	if err != nil {
		return fmt.Errorf("RedisRepo - DeleteAllRefreshTokens - SMembers: %w", err)
	}

	if len(hashes) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	for _, hash := range hashes {
		pipe.Del(ctx, refreshTokenPrefix+hash)
	}
	pipe.Del(ctx, userTokensPrefix+userID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("RedisRepo - DeleteAllRefreshTokens - pipe.Exec: %w", err)
	}
	return nil
}
