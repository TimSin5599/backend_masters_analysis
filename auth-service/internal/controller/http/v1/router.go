package v1

import (
	"auth-service/internal/entity"
	"auth-service/internal/usecase"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	_ "auth-service/docs"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type authRoutes struct {
	t usecase.Auth
}

// LoggingMiddleware - middleware для подробного логирования всех HTTP запросов
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Время начала запроса
		startTime := time.Now()

		// Логируем входящий запрос
		fmt.Printf("\n[ЗАПРОС] %s | Метод: %s | Путь: %s | IP: %s | User-Agent: %s\n",
			startTime.Format("2006-01-02 15:04:05"),
			c.Request.Method,
			c.Request.URL.Path,
			c.ClientIP(),
			c.Request.UserAgent(),
		)

		// Обрабатываем запрос
		c.Next()

		// Время окончания запроса
		duration := time.Since(startTime)

		// Логируем ответ
		fmt.Printf("[ОТВЕТ] %s | Статус: %d | Время обработки: %v | Путь: %s\n",
			time.Now().Format("2006-01-02 15:04:05"),
			c.Writer.Status(),
			duration,
			c.Request.URL.Path,
		)
	}
}

func NewRouter(handler *gin.Engine, t usecase.Auth, jwtSecret string) {
	// Middleware для логирования
	handler.Use(LoggingMiddleware())

	// CORS настройки
	handler.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	routes := &authRoutes{t}
	
	// Эндпоинты под префиксом /v1
	v1 := handler.Group("/v1")
	{
		v1.POST("/login", routes.login)
		v1.POST("/register", JWTMiddleware(jwtSecret, t), AdminMiddleware(), routes.register)

		// Защищенные эндпоинты (требуют JWT токен)
		auth := v1.Group("/auth")
		auth.Use(JWTMiddleware(jwtSecret, t))
		{
			auth.GET("/me", routes.getCurrentUser)
			auth.POST("/change-password", routes.changePassword)
		}

		users := v1.Group("/users")
		users.Use(JWTMiddleware(jwtSecret, t))
		{
			users.POST("", AdminMiddleware(), routes.createUser)
			users.GET("", routes.listUsers)
			users.GET("/", routes.listUsers)
			users.PUT("/:id", routes.updateUser)
			users.DELETE("/:id", AdminMiddleware(), routes.deleteUser)
			users.GET("/:id", routes.getUserByID)
		}
	}

	fmt.Println("\n=== ЗАРЕГИСТРИРОВАННЫЕ ЭНДПОИНТЫ ===")
	fmt.Println("Публичные:")
	fmt.Println("  POST /v1/register - Регистрация нового пользователя")
	fmt.Println("  POST /v1/login    - Авторизация пользователя")
	fmt.Println("Защищенные (требуют JWT):")
	fmt.Println("  GET  /auth/me     - Получение текущего пользователя")
	fmt.Println("  GET  /users/:id   - Получение пользователя по ID")
	fmt.Println("Swagger UI:")
	fmt.Println("  GET  /swagger/*any - Swagger документация")
	fmt.Println("=====================================")

	// Swagger UI
	handler.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

// @Summary Регистрация
// @Description Регистрация нового пользователя (только для администраторов)
// @Tags auth
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body registerRequest true "Данные для регистрации"
// @Success 200 {object} entity.User
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/register [post]
func (r *authRoutes) register(c *gin.Context) {
	fmt.Printf("\n[REGISTER] Начало обработки запроса регистрации\n")

	var request registerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fmt.Printf("[REGISTER] ❌ Ошибка валидации данных: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("[REGISTER] ✓ Данные валидированы успешно\n")
	fmt.Printf("[REGISTER] Email: %s | Роль: %s\n", request.Email, request.Role)

	user, err := r.t.Register(c.Request.Context(), request.Email, request.Password, request.Role)
	if err != nil {
		fmt.Printf("[REGISTER] ❌ Ошибка регистрации пользователя: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("[REGISTER] ✅ Пользователь успешно зарегистрирован | ID: %s | Email: %s\n", user.ID, user.Email)
	c.JSON(http.StatusOK, user)
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// @Summary Авторизация
// @Description Вход в систему и получение JWT токена
// @Tags auth
// @Accept json
// @Produce json
// @Param request body loginRequest true "Данные для входа"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /v1/login [post]
func (r *authRoutes) login(c *gin.Context) {
	fmt.Printf("\n[LOGIN] Начало обработки запроса авторизации\n")

	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fmt.Printf("[LOGIN] ❌ Ошибка валидации данных (Bad Request): %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	fmt.Printf("[LOGIN] ✓ Данные валидированы успешно\n")
	fmt.Printf("[LOGIN] Email: %s\n", request.Email)

	token, err := r.t.Login(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		fmt.Printf("[LOGIN] ❌ Ошибка авторизации: %v\n", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("[LOGIN] ✅ Пользователь успешно авторизован | Email: %s | Токен сгенерирован\n", request.Email)
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// @Summary Текущий пользователь
// @Description Получение информации о текущем авторизованном пользователе
// @Tags auth
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/auth/me [get]
func (r *authRoutes) getCurrentUser(c *gin.Context) {
	fmt.Printf("\n[GET_ME] Получение информации о текущем пользователе\n")

	// Получаем userID из контекста (установлен JWT middleware)
	userID, exists := c.Get("userID")
	if !exists {
		fmt.Printf("[GET_ME] ❌ UserID не найден в контексте\n")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	fmt.Printf("[GET_ME] UserID из токена: %s\n", userID)

	// Получаем пользователя из базы
	user, err := r.t.GetByID(c.Request.Context(), userID.(string))
	if err != nil {
		fmt.Printf("[GET_ME] ❌ Ошибка получения пользователя: %v\n", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	fmt.Printf("[GET_ME] ✅ Пользователь найден | Email: %s | Role: %s\n", user.Email, user.Role)

	// Не возвращаем пароль клиенту
	response := gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"role":       user.Role,
		"firstName":  user.FirstName,
		"lastName":   user.LastName,
		"phone":      user.Phone,
		"avatarPath": user.AvatarPath,
		"createdAt":  user.CreatedAt,
		"updatedAt":  user.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Пользователь по ID
// @Description Получение информации о пользователе по его ID
// @Tags users
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "ID пользователя"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/users/{id} [get]
func (r *authRoutes) getUserByID(c *gin.Context) {
	userID := c.Param("id")
	fmt.Printf("\n[GET_USER_BY_ID] Получение пользователя по ID: %s\n", userID)

	// Проверяем валидность ID
	if userID == "" || userID == "null" {
		fmt.Printf("[GET_USER_BY_ID] ❌ Некорректный ID пользователя\n")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Получаем пользователя из базы
	user, err := r.t.GetByID(c.Request.Context(), userID)
	if err != nil {
		fmt.Printf("[GET_USER_BY_ID] ❌ Ошибка получения пользователя: %v\n", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	fmt.Printf("[GET_USER_BY_ID] ✅ Пользователь найден | Email: %s | Role: %s\n", user.Email, user.Role)

	// Не возвращаем пароль клиенту
	response := gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"role":       user.Role,
		"firstName":  user.FirstName,
		"lastName":   user.LastName,
		"phone":      user.Phone,
		"avatarPath": user.AvatarPath,
		"createdAt":  user.CreatedAt,
		"updatedAt":  user.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Создать пользователя
// @Description Создание нового пользователя администратором
// @Tags users
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body entity.User true "Данные пользователя"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/users [post]
func (r *authRoutes) createUser(c *gin.Context) {
	fmt.Printf("\n[CREATE_USER] Создание нового пользователя\n")

	var userReq entity.User
	if err := c.ShouldBindJSON(&userReq); err != nil {
		fmt.Printf("[CREATE_USER] ❌ Ошибка валидации данных: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := r.t.CreateUser(c.Request.Context(), userReq)
	if err != nil {
		fmt.Printf("[CREATE_USER] ❌ Ошибка создания пользователя: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "User created successfully"})
}

// @Summary Список пользователей
// @Description Получение списка всех пользователей
// @Tags users
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /v1/users [get]
func (r *authRoutes) listUsers(c *gin.Context) {
	fmt.Printf("\n[LIST_USERS] Получение списка всех пользователей\n")

	users, err := r.t.ListUsers(c.Request.Context())
	if err != nil {
		fmt.Printf("[LIST_USERS] ❌ Ошибка получения списка пользователей: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Формируем ответ без паролей
	type userResponse struct {
		ID         string    `json:"id"`
		Email      string    `json:"email"`
		FirstName  string    `json:"firstName"`
		LastName   string    `json:"lastName"`
		Phone      string    `json:"phone"`
		AvatarPath string    `json:"avatarPath"`
		Role       string    `json:"role"`
		LastOnline time.Time `json:"lastOnline"`
		CreatedAt  time.Time `json:"createdAt"`
		UpdatedAt  time.Time `json:"updatedAt"`
	}

	rows := make([]userResponse, 0, len(users))
	for _, u := range users {
		rows = append(rows, userResponse{
			ID:         u.ID,
			Email:      u.Email,
			FirstName:  u.FirstName,
			LastName:   u.LastName,
			Phone:      u.Phone,
			AvatarPath: u.AvatarPath,
			Role:       u.Role,
			LastOnline: u.LastOnline,
			CreatedAt:  u.CreatedAt,
			UpdatedAt:  u.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"rows":  rows,
		"count": len(rows),
	})
}

// @Summary Обновить пользователя
// @Description Обновление данных пользователя по ID
// @Tags users
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "ID пользователя"
// @Param request body entity.User true "Новые данные пользователя"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/users/{id} [put]
func (r *authRoutes) updateUser(c *gin.Context) {
	id := c.Param("id")
	fmt.Printf("\n[UPDATE_USER] Обновление пользователя ID: %s\n", id)

	var userReq entity.User
	if err := c.ShouldBindJSON(&userReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userReq.ID = id
	err := r.t.UpdateUser(c.Request.Context(), userReq)
	if err != nil {
		fmt.Printf("[UPDATE_USER] ❌ Ошибка обновления: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

type changePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// @Summary Сменить пароль
// @Description Изменение пароля текущего пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body changePasswordRequest true "Старый и новый пароли"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /v1/auth/change-password [post]
func (r *authRoutes) changePassword(c *gin.Context) {
	fmt.Printf("\n[CHANGE_PASSWORD] Начало обработки запроса смены пароля\n")

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := r.t.ChangePassword(c.Request.Context(), userID.(string), req.OldPassword, req.NewPassword)
	if err != nil {
		fmt.Printf("[CHANGE_PASSWORD] ❌ Ошибка: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("[CHANGE_PASSWORD] ✅ Пароль успешно изменен для userID: %s\n", userID)
	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// @Summary Удаление пользователя
// @Description Удаляет пользователя по ID (только администраторы)
// @Tags auth
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "User ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/users/{id} [delete]
func (r *authRoutes) deleteUser(c *gin.Context) {
	id := c.Param("id")
	fmt.Printf("\n[AUTH_ROUTER] Начало обработки удаления пользователя: %s\n", id)

	err := r.t.DeleteUser(c.Request.Context(), id)
	if err != nil {
		fmt.Printf("[AUTH_ROUTER] ❌ Ошибка удаления пользователя %s: %v\n", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("[AUTH_ROUTER] ✅ Пользователь %s успешно удален\n", id)
	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
