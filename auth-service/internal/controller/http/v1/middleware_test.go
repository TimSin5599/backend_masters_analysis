package v1_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	v1 "auth-service/internal/controller/http/v1"
	"auth-service/internal/domain/entity"
	pkgjwt "auth-service/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

const testSecret = "test-secret-key"

// ─── Mock User UseCase for middleware ────────────────────────────────────────

type middlewareMockUserUC struct{}

func (m *middlewareMockUserUC) CreateUser(ctx context.Context, user entity.User) (string, error) {
	return "", nil
}
func (m *middlewareMockUserUC) GetByID(ctx context.Context, id string) (entity.User, error) {
	return entity.User{ID: id}, nil
}
func (m *middlewareMockUserUC) ListUsers(ctx context.Context) ([]entity.User, error) {
	return nil, nil
}
func (m *middlewareMockUserUC) UpdateUser(ctx context.Context, user entity.User) error { return nil }
func (m *middlewareMockUserUC) UpdateLastOnline(ctx context.Context, id string) error  { return nil }
func (m *middlewareMockUserUC) DeleteUser(ctx context.Context, id string) error        { return nil }

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newMiddlewareRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mw := v1.NewMiddleware(testSecret, &middlewareMockUserUC{})

	protected := r.Group("/protected")
	protected.Use(mw.JWTMiddleware(testSecret, &middlewareMockUserUC{}))
	protected.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"userID": c.GetString("userID")})
	})

	adminOnly := r.Group("/admin")
	adminOnly.Use(mw.JWTMiddleware(testSecret, &middlewareMockUserUC{}))
	adminOnly.Use(mw.AdminMiddleware())
	adminOnly.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	return r
}

func makeToken(roles []string, secret string) string {
	token, _ := pkgjwt.GenerateAccessToken("uid1", "u@test.com", roles, secret)
	return token
}

// ─── JWTMiddleware ────────────────────────────────────────────────────────────

func TestJWTMiddleware_NoHeader(t *testing.T) {
	r := newMiddlewareRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected/ping", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTMiddleware_BadFormat(t *testing.T) {
	r := newMiddlewareRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected/ping", nil)
	req.Header.Set("Authorization", "InvalidToken")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTMiddleware_WrongSecret(t *testing.T) {
	token := makeToken([]string{entity.RoleAdmin}, "other-secret")
	r := newMiddlewareRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTMiddleware_ValidToken(t *testing.T) {
	token := makeToken([]string{entity.RoleAdmin}, testSecret)
	r := newMiddlewareRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ─── AdminMiddleware ──────────────────────────────────────────────────────────

func TestAdminMiddleware_NonAdmin(t *testing.T) {
	token := makeToken([]string{entity.RoleExpert}, testSecret)
	r := newMiddlewareRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAdminMiddleware_Admin(t *testing.T) {
	token := makeToken([]string{entity.RoleAdmin}, testSecret)
	r := newMiddlewareRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ─── Expired token ────────────────────────────────────────────────────────────

func TestJWTMiddleware_ExpiredToken(t *testing.T) {
	// Pre-signed expired JWT (HS256, exp=1 i.e. 1970, signed with testSecret).
	expiredToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
		"eyJleHAiOjEsImlhdCI6MSwic3ViIjoidWlkMSIsInVzZXIiOnsiZW1haWwiOiJ1QHRlc3QuY29tIiwicm9sZXMiOlsiYWRtaW4iXX19." +
		"signature-placeholder"

	r := newMiddlewareRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected/ping", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
