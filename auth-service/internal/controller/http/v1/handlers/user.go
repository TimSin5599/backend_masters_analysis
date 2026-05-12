package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"auth-service/internal/domain"
	"auth-service/internal/domain/entity"
	"auth-service/internal/usecase"
)

type userHandler struct {
	uc usecase.User
}

type userResponse struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	FirstName  string    `json:"firstName"`
	LastName   string    `json:"lastName"`
	Phone      string    `json:"phone"`
	AvatarPath string    `json:"avatarPath"`
	Roles      []string  `json:"roles"`
	LastOnline time.Time `json:"lastOnline"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type createUserRequest struct {
	Email     string   `json:"email"    binding:"required,email"`
	Password  string   `json:"password" binding:"required"`
	FirstName string   `json:"firstName"`
	LastName  string   `json:"lastName"`
	Phone     string   `json:"phone"`
	Roles     []string `json:"roles"    binding:"required"`
}

type updateUserRequest struct {
	FirstName  string   `json:"firstName"`
	LastName   string   `json:"lastName"`
	Phone      string   `json:"phone"`
	AvatarPath string   `json:"avatarPath"`
	Roles      []string `json:"roles"`
}

func NewUserHandler(uc usecase.User) *userHandler {
	return &userHandler{uc: uc}
}

// @Summary Текущий пользователь
// @Description Получение информации о текущем авторизованном пользователе
// @Tags auth
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Router /v1/auth/me [get]
func (r *userHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, err := r.uc.GetByID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// @Summary Пользователь по ID
// @Description Получение информации о пользователе по его ID
// @Tags users
// @Security ApiKeyAuth
// @Param id path string true "ID пользователя"
// @Success 200 {object} map[string]interface{}
// @Router /v1/users/{id} [get]
func (r *userHandler) GetUserByID(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" || userID == "null" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := r.uc.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// @Summary Создать пользователя
// @Tags users
// @Security ApiKeyAuth
// @Success 201 {object} map[string]interface{}
// @Router /v1/users [post]
func (r *userHandler) CreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	user := entity.User{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		Roles:     req.Roles,
	}

	id, err := r.uc.CreateUser(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "User created successfully"})
}

// @Summary Список пользователей
// @Tags users
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Router /v1/users [get]
func (r *userHandler) ListUsers(c *gin.Context) {
	users, err := r.uc.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows := make([]userResponse, 0, len(users))
	for _, u := range users {
		rows = append(rows, toUserResponse(u))
	}

	c.JSON(http.StatusOK, gin.H{"rows": rows, "count": len(rows)})
}

// @Summary Обновить пользователя
// @Tags users
// @Security ApiKeyAuth
// @Router /v1/users/{id} [put]
func (r *userHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := entity.User{
		ID:         id,
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		Phone:      req.Phone,
		AvatarPath: req.AvatarPath,
		Roles:      req.Roles,
	}

	if err := r.uc.UpdateUser(c.Request.Context(), user); err != nil {
		switch err {
		case domain.ErrUserNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// @Summary Удаление пользователя
// @Tags auth
// @Security ApiKeyAuth
// @Router /v1/users/{id} [delete]
func (r *userHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := r.uc.DeleteUser(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func toUserResponse(u entity.User) userResponse {
	return userResponse{
		ID:         u.ID,
		Email:      u.Email,
		FirstName:  u.FirstName,
		LastName:   u.LastName,
		Phone:      u.Phone,
		AvatarPath: u.AvatarPath,
		Roles:      u.Roles,
		LastOnline: u.LastOnline,
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
	}
}
