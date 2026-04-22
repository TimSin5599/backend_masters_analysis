package v1

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	lastOnlineUpdates = make(map[string]time.Time)
	mu                sync.Mutex
)

// LoggingMiddleware — middleware для логирования HTTP запросов.
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		fmt.Printf("\n[MANAGE] %s | %s %s | IP: %s\n",
			startTime.Format("2006-01-02 15:04:05"),
			c.Request.Method,
			c.Request.URL.Path,
			c.ClientIP(),
		)
		c.Next()
		fmt.Printf("[MANAGE] Статус: %d | Время: %v\n", c.Writer.Status(), time.Since(startTime))
	}
}

// JWTMiddleware проверяет access токен из заголовка Authorization: Bearer <token>.
func JWTMiddleware(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		var tokenString string

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		if tokenString == "" {
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secretKey), nil
		})

		if err != nil {
			errMsg := "invalid token"
			if strings.Contains(err.Error(), "token is expired") {
				errMsg = "token expired"
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": errMsg})
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		userID, ok := claims["sub"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token: missing user ID"})
			c.Abort()
			return
		}

		// Throttled last_online update — не блокируем запрос
		mu.Lock()
		if lastUpdate, exists := lastOnlineUpdates[userID]; !exists || time.Since(lastUpdate) > 5*time.Minute {
			lastOnlineUpdates[userID] = time.Now()
			go func() {
				// manage-service не имеет доступа к auth repo — просто помечаем локально
				_ = context.Background()
			}()
		}
		mu.Unlock()

		c.Set("userID", userID)
		c.Set("claims", claims)
		c.Next()
	}
}

// roleFromClaims извлекает роль из JWT claims.
// Auth-service кладёт роль в claims["user"]["role"].
func roleFromClaims(c *gin.Context) (string, bool) {
	raw, exists := c.Get("claims")
	if !exists {
		return "", false
	}
	mapClaims, ok := raw.(jwt.MapClaims)
	if !ok {
		return "", false
	}
	// Основной путь: claims["user"]["role"]
	if userMap, ok := mapClaims["user"].(map[string]interface{}); ok {
		if role, ok := userMap["role"].(string); ok && role != "" {
			return role, true
		}
	}
	// Запасной путь: claims["role"] (на случай другого формата токена)
	if role, ok := mapClaims["role"].(string); ok && role != "" {
		return role, true
	}
	return "", false
}

// AdminMiddleware разрешает доступ только пользователям с ролью admin.
// Должен применяться после JWTMiddleware.
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := roleFromClaims(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		if role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// AdminOrManagerMiddleware разрешает доступ пользователям с ролью admin или manager.
func AdminOrManagerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := roleFromClaims(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		if role != "admin" && role != "manager" {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin or manager access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// NoCacheMiddleware добавляет заголовки для отключения кэширования в браузере
func NoCacheMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Next()
	}
}
