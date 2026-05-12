package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"auth-service/internal/controller/http/v1/handlers"
	"auth-service/internal/domain"
	"auth-service/internal/domain/entity"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Mock Auth UseCase ────────────────────────────────────────────────────────

type mockAuthUC struct {
	LoginFunc          func(ctx context.Context, email, password string) (string, string, error)
	RefreshTokensFunc  func(ctx context.Context, refreshToken string) (string, string, error)
	LogoutFunc         func(ctx context.Context, refreshToken string) error
	ChangePasswordFunc func(ctx context.Context, userID, oldPassword, newPassword string) error
}

func (m *mockAuthUC) Register(ctx context.Context, email, password, role string) (entity.User, error) {
	return entity.User{}, nil
}
func (m *mockAuthUC) Login(ctx context.Context, email, password string) (string, string, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, email, password)
	}
	return "", "", nil
}
func (m *mockAuthUC) RefreshTokens(ctx context.Context, refreshToken string) (string, string, error) {
	if m.RefreshTokensFunc != nil {
		return m.RefreshTokensFunc(ctx, refreshToken)
	}
	return "", "", nil
}
func (m *mockAuthUC) Logout(ctx context.Context, refreshToken string) error {
	if m.LogoutFunc != nil {
		return m.LogoutFunc(ctx, refreshToken)
	}
	return nil
}
func (m *mockAuthUC) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	if m.ChangePasswordFunc != nil {
		return m.ChangePasswordFunc(ctx, userID, oldPassword, newPassword)
	}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newAuthRouter(uc *mockAuthUC) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handlers.NewAuthHandler(uc)
	r.POST("/v1/login", h.Login)
	r.POST("/v1/refresh", h.Refresh)
	r.POST("/v1/logout", h.Logout)
	// ChangePassword requires userID in context — inject via middleware in test
	r.POST("/v1/auth/change-password", func(c *gin.Context) {
		c.Set("userID", "uid1")
		h.ChangePassword(c)
	})
	return r
}

func jsonBody(v interface{}) *bytes.Buffer {
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

// ─── Login ────────────────────────────────────────────────────────────────────

func TestAuthHandler_Login_BadBody(t *testing.T) {
	r := newAuthRouter(&mockAuthUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	uc := &mockAuthUC{
		LoginFunc: func(ctx context.Context, email, password string) (string, string, error) {
			return "", "", domain.ErrInvalidCredentials
		},
	}
	r := newAuthRouter(uc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/login",
		jsonBody(map[string]string{"email": "a@b.com", "password": "wrong"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	uc := &mockAuthUC{
		LoginFunc: func(ctx context.Context, email, password string) (string, string, error) {
			return "access-token", "refresh-token", nil
		},
	}
	r := newAuthRouter(uc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/login",
		jsonBody(map[string]string{"email": "a@b.com", "password": "pass"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "access-token", resp["access_token"])
}

// ─── Refresh ──────────────────────────────────────────────────────────────────

func TestAuthHandler_Refresh_NoCookie(t *testing.T) {
	r := newAuthRouter(&mockAuthUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/refresh", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	uc := &mockAuthUC{
		RefreshTokensFunc: func(ctx context.Context, refreshToken string) (string, string, error) {
			return "", "", domain.ErrInvalidToken
		},
	}
	r := newAuthRouter(uc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "bad-token"})
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Refresh_Success(t *testing.T) {
	uc := &mockAuthUC{
		RefreshTokensFunc: func(ctx context.Context, refreshToken string) (string, string, error) {
			return "new-access", "new-refresh", nil
		},
	}
	r := newAuthRouter(uc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "valid-token"})
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "new-access", resp["access_token"])
}

// ─── Logout ───────────────────────────────────────────────────────────────────

func TestAuthHandler_Logout_AlwaysOK(t *testing.T) {
	r := newAuthRouter(&mockAuthUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/logout", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ─── ChangePassword ───────────────────────────────────────────────────────────

func TestAuthHandler_ChangePassword_BadBody(t *testing.T) {
	r := newAuthRouter(&mockAuthUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/change-password", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_ChangePassword_InvalidCredentials(t *testing.T) {
	uc := &mockAuthUC{
		ChangePasswordFunc: func(ctx context.Context, userID, oldPassword, newPassword string) error {
			return domain.ErrInvalidCredentials
		},
	}
	r := newAuthRouter(uc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/change-password",
		jsonBody(map[string]string{"oldPassword": "old", "newPassword": "new"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_ChangePassword_Success(t *testing.T) {
	r := newAuthRouter(&mockAuthUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/change-password",
		jsonBody(map[string]string{"oldPassword": "old", "newPassword": "new"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
