package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenTTL  = 20 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour
)

// GenerateAccessToken генерирует JWT access токен.
func GenerateAccessToken(userID, email, role, secretKey string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(AccessTokenTTL).Unix(),
		"iat": time.Now().Unix(),
		"user": map[string]interface{}{
			"email": email,
			"role":  role,
		},
	})
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("AuthUseCase - generateAccessToken - token.SignedString: %w", err)
	}
	return tokenString, nil
}

// GenerateRefreshToken генерирует JWT refresh токен.
func GenerateRefreshToken(userID, secretKey string) (string, error) {
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  userID,
		"exp":  time.Now().Add(RefreshTokenTTL).Unix(),
		"iat":  time.Now().Unix(),
		"type": "refresh",
	})

	refreshTokenString, err := refreshToken.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return refreshTokenString, nil
}

// ValidateToken проверяет строку токена и возвращает его payload (claims),
// если он имеет валидную подпись и не истёк.
func ValidateToken(tokenString, secretKey string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Проверяем метод хеширования
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
