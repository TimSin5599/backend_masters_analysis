package v1

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"manage-service/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	lastOnlineUpdates = make(map[string]time.Time)
	mu                sync.Mutex
)

// LoggingMiddleware — middleware для логирования и сбора Prometheus HTTP-метрик.
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

		duration := time.Since(startTime).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		metrics.HttpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		metrics.HttpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)

		fmt.Printf("[MANAGE] Статус: %s | Время: %v\n", status, time.Since(startTime))
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

// rolesFromClaims извлекает массив ролей из JWT claims["user"]["roles"].
func rolesFromClaims(c *gin.Context) ([]string, bool) {
	raw, exists := c.Get("claims")
	if !exists {
		return nil, false
	}
	mapClaims, ok := raw.(jwt.MapClaims)
	if !ok {
		return nil, false
	}
	userMap, ok := mapClaims["user"].(map[string]interface{})
	if !ok {
		return nil, false
	}
	switch v := userMap["roles"].(type) {
	case []interface{}:
		roles := make([]string, 0, len(v))
		for _, r := range v {
			if s, ok := r.(string); ok {
				roles = append(roles, s)
			}
		}
		return roles, len(roles) > 0
	case []string:
		return v, len(v) > 0
	}
	return nil, false
}

func hasRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// AdminMiddleware разрешает доступ только пользователям с ролью admin.
// Должен применяться после JWTMiddleware.
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, ok := rolesFromClaims(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		if !hasRole(roles, "admin") {
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
		roles, ok := rolesFromClaims(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		if !hasRole(roles, "admin") && !hasRole(roles, "manager") {
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
