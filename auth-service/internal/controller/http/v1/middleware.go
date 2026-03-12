package v1

import (
	"auth-service/internal/entity"
	"auth-service/internal/usecase"
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

// JWTMiddleware создает middleware для проверки JWT токена
func JWTMiddleware(secretKey string, authUC usecase.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Printf("\n[JWT_MIDDLEWARE] Проверка авторизации для %s %s\n", c.Request.Method, c.Request.URL.Path)

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			fmt.Printf("[JWT_MIDDLEWARE] ❌ Заголовок Authorization отсутствует\n")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Проверяем формат "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			fmt.Printf("[JWT_MIDDLEWARE] ❌ Неверный формат заголовка Authorization\n")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		fmt.Printf("[JWT_MIDDLEWARE] Токен получен, проверяем подпись...\n")

		// Парсим и проверяем токен
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Проверяем метод подписи
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secretKey), nil
		})

		if err != nil {
			fmt.Printf("[JWT_MIDDLEWARE] ❌ Ошибка парсинга токена: %v\n", err)
			if strings.Contains(err.Error(), "token is expired") {
				fmt.Printf("[JWT_MIDDLEWARE] ❌ Токен просрочен\n")
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			c.Abort()
			return
		}

		if !token.Valid {
			fmt.Printf("[JWT_MIDDLEWARE] ❌ Токен недействителен (Valid == false)\n")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token (validity check failed)"})
			c.Abort()
			return
		}

		// Извлекаем claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			fmt.Printf("[JWT_MIDDLEWARE] ❌ Не удалось извлечь claims\n")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Получаем user ID из sub
		userID, ok := claims["sub"].(string)
		if !ok {
			fmt.Printf("[JWT_MIDDLEWARE] ❌ Отсутствует поле 'sub' в токене\n")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: missing user ID"})
			c.Abort()
			return
		}

		fmt.Printf("[JWT_MIDDLEWARE] ✅ Токен валиден | UserID: %s\n", userID)

		// Throttled last_online update
		mu.Lock()
		lastUpdate, exists := lastOnlineUpdates[userID]
		if !exists || time.Since(lastUpdate) > 5*time.Minute {
			go func(uid string) {
				_ = authUC.UpdateLastOnline(context.Background(), uid)
			}(userID)
			lastOnlineUpdates[userID] = time.Now()
		}
		mu.Unlock()

		// Сохраняем userID в контексте для дальнейшего использования
		c.Set("userID", userID)
		c.Set("claims", claims)

		c.Next()
	}
}

// AdminMiddleware проверяет наличие роли admin у текущего пользователя
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Printf("\n[ADMIN_MIDDLEWARE] Проверка прав администратора для %s %s\n", c.Request.Method, c.Request.URL.Path)

		claimsInterface, exists := c.Get("claims")
		if !exists {
			fmt.Printf("[ADMIN_MIDDLEWARE] ❌ Claims не найдены в контексте\n")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		claims, ok := claimsInterface.(jwt.MapClaims)
		if !ok {
			fmt.Printf("[ADMIN_MIDDLEWARE] ❌ Неверный формат claims\n")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid claims format"})
			c.Abort()
			return
		}

		userClaim, ok := claims["user"].(map[string]interface{})
		if !ok {
			fmt.Printf("[ADMIN_MIDDLEWARE] ❌ Данные пользователя не найдены в токене\n")
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: incomplete token data"})
			c.Abort()
			return
		}

		role, ok := userClaim["role"].(string)
		if !ok || role != entity.RoleAdmin {
			fmt.Printf("[ADMIN_MIDDLEWARE] ❌ Отказ в доступе. Требуется admin, текущая роль: %v\n", role)
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Requires admin role"})
			c.Abort()
			return
		}

		fmt.Printf("[ADMIN_MIDDLEWARE] ✅ Права администратора подтверждены\n")
		c.Next()
	}
}
