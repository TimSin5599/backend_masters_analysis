package handlers

import (
	"net/http"
	"time"

	"auth-service/internal/domain"
	"auth-service/internal/usecase"
	"auth-service/pkg/metrics"

	"github.com/gin-gonic/gin"
)

// ─── Request types ──────────────────────────────────────────────────────────

const refreshCookieName = "refresh_token"

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type changePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

type authHandler struct {
	uc usecase.Auth
}

func NewAuthHandler(uc usecase.Auth) *authHandler {
	return &authHandler{uc: uc}
}

// @Summary Авторизация
// @Description Вход в систему. Возвращает access_token в JSON и устанавливает refresh_token в httpOnly cookie.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body loginRequest true "Данные для входа"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /v1/login [post]
func (r *authHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	accessToken, refreshToken, err := r.uc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		switch err {
		case domain.ErrInvalidCredentials:
			metrics.LoginAttemptsTotal.WithLabelValues("invalid_credentials").Inc()
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		case domain.ErrInvalidToken:
			metrics.LoginAttemptsTotal.WithLabelValues("internal_error").Inc()
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token generation"})
		case domain.ErrTokenRotation:
			metrics.LoginAttemptsTotal.WithLabelValues("internal_error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Token rotation failed"})
		default:
			metrics.LoginAttemptsTotal.WithLabelValues("internal_error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	metrics.LoginAttemptsTotal.WithLabelValues("success").Inc()

	// Refresh токен — в httpOnly cookie (недоступна JS)
	c.SetCookie(
		refreshCookieName,
		refreshToken,
		int(7*24*time.Hour/time.Second), // MaxAge в секундах
		"/api/auth/v1/",                 // Path совпадает с nginx-префиксом
		"",                              // Domain (пусто = текущий хост)
		true,                            // Secure — только HTTPS
		true,                            // HttpOnly
	)

	c.JSON(http.StatusOK, gin.H{"access_token": accessToken})
}

// @Summary Обновить токены
// @Description Обновляет access токен по refresh токену из httpOnly cookie. Реализует ротацию refresh токена.
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /v1/refresh [post]
func (r *authHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(refreshCookieName)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token cookie missing"})
		return
	}

	newAccessToken, newRefreshToken, err := r.uc.RefreshTokens(c.Request.Context(), refreshToken)
	if err != nil {
		// Сбрасываем cookie при невалидном токене
		c.SetCookie(refreshCookieName, "", -1, "/api/auth/v1/", "", true, true)
		switch err {
		case domain.ErrInvalidToken:
			metrics.TokenRefreshTotal.WithLabelValues("invalid_token").Inc()
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
		case domain.ErrTokenRotation:
			metrics.TokenRefreshTotal.WithLabelValues("internal_error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Token rotation failed"})
		default:
			metrics.TokenRefreshTotal.WithLabelValues("internal_error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	metrics.TokenRefreshTotal.WithLabelValues("success").Inc()

	// Устанавливаем новый refresh cookie (ротация)
	c.SetCookie(
		refreshCookieName,
		newRefreshToken,
		int(7*24*time.Hour/time.Second),
		"/api/auth/v1/",
		"",
		true,
		true,
	)

	c.JSON(http.StatusOK, gin.H{"access_token": newAccessToken})
}

// @Summary Выход
// @Description Инвалидирует refresh токен и очищает cookie.
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]string
// @Router /v1/logout [post]
func (r *authHandler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie(refreshCookieName)
	if err == nil && refreshToken != "" {
		// Ошибку игнорируем — логаут всегда успешный с точки зрения клиента
		_ = r.uc.Logout(c.Request.Context(), refreshToken)
	}

	// Очищаем cookie
	c.SetCookie(refreshCookieName, "", -1, "/api/auth/v1/", "", true, true)
	metrics.LogoutTotal.Inc()
	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// @Summary Сменить пароль
// @Tags auth
// @Security ApiKeyAuth
// @Router /v1/auth/change-password [post]
func (r *authHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if err := r.uc.ChangePassword(c.Request.Context(), userID.(string), req.OldPassword, req.NewPassword); err != nil {
		switch err {
		case domain.ErrUserNotFound:
			metrics.PasswordChangesTotal.WithLabelValues("not_found").Inc()
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		case domain.ErrInvalidCredentials:
			metrics.PasswordChangesTotal.WithLabelValues("invalid_credentials").Inc()
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid old password"})
		case domain.ErrPasswordChange:
			metrics.PasswordChangesTotal.WithLabelValues("internal_error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Password change failed"})
		case domain.ErrTokenRotation:
			metrics.PasswordChangesTotal.WithLabelValues("internal_error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Token invalidation failed"})
		default:
			metrics.PasswordChangesTotal.WithLabelValues("internal_error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	metrics.PasswordChangesTotal.WithLabelValues("success").Inc()

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}
