package v1

import (
	"fmt"
	"strings"
	"time"

	"auth-service/internal/usecase"
	"auth-service/internal/controller/http/v1/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	_ "auth-service/docs"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewRouter(handler *gin.Engine, t usecase.Auth, u usecase.User, jwtSecret string, corsOrigin string) {
	mw := NewMiddleware(jwtSecret, u)
	
	handler.Use(mw.LoggingMiddleware())

	// Для работы с httpOnly cookie нужен конкретный origin (не "*") и AllowCredentials: true
	allowOrigin := corsOrigin
	if allowOrigin == "" {
		allowOrigin = "http://localhost:3000"
	}
	allowedOrigins := strings.Split(allowOrigin, ",")

	handler.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			for _, o := range allowedOrigins {
				if origin == strings.TrimSpace(o) {
					return true
				}
			}
			return isLocalNetworkOrigin(origin)
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	authHandler := handlers.NewAuthHandler(t)
    userHandler := handlers.NewUserHandler(u)

	v1 := handler.Group("/v1")
	{
		// Публичные эндпоинты
		v1.POST("/login", authHandler.Login)
		v1.POST("/refresh", authHandler.Refresh)
		v1.POST("/logout", authHandler.Logout)

		// Защищённые эндпоинты
		auth := v1.Group("/auth")
		auth.Use(mw.JWTMiddleware(jwtSecret, u))
		{
			auth.GET("/me", userHandler.GetCurrentUser)
			auth.POST("/change-password", authHandler.ChangePassword)
		}

		users := v1.Group("/users")
		users.Use(mw.JWTMiddleware(jwtSecret, u))
		{
			users.POST("", mw.AdminMiddleware(), userHandler.CreateUser)
			users.GET("", userHandler.ListUsers)
			users.GET("/", userHandler.ListUsers)
			users.PUT("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", mw.AdminMiddleware(), userHandler.DeleteUser)
			users.GET("/:id", userHandler.GetUserByID)
		}
	}

	handler.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	fmt.Println("\n=== ЗАРЕГИСТРИРОВАННЫЕ ЭНДПОИНТЫ ===")
	fmt.Println("Публичные:")
	fmt.Println("  POST /v1/login    - Авторизация (access+refresh)")
	fmt.Println("  POST /v1/refresh  - Обновление access токена по httpOnly cookie")
	fmt.Println("  POST /v1/logout   - Инвалидация refresh токена")
	fmt.Println("Защищённые (требуют JWT):")
	fmt.Println("  GET  /v1/auth/me  - Текущий пользователь")
	fmt.Printf("CORS origin: %s\n", allowOrigin)
	fmt.Println("=====================================")
}
