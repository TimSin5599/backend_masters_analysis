package handlers_test

import (
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
)

// ─── Mock User UseCase ────────────────────────────────────────────────────────

type mockUserUC struct {
	CreateUserFunc       func(ctx context.Context, user entity.User) (string, error)
	GetByIDFunc          func(ctx context.Context, id string) (entity.User, error)
	ListUsersFunc        func(ctx context.Context) ([]entity.User, error)
	UpdateUserFunc       func(ctx context.Context, user entity.User) error
	UpdateLastOnlineFunc func(ctx context.Context, id string) error
	DeleteUserFunc       func(ctx context.Context, id string) error
}

func (m *mockUserUC) CreateUser(ctx context.Context, user entity.User) (string, error) {
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(ctx, user)
	}
	return "new-id", nil
}
func (m *mockUserUC) GetByID(ctx context.Context, id string) (entity.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return entity.User{}, domain.ErrUserNotFound
}
func (m *mockUserUC) ListUsers(ctx context.Context) ([]entity.User, error) {
	if m.ListUsersFunc != nil {
		return m.ListUsersFunc(ctx)
	}
	return nil, nil
}
func (m *mockUserUC) UpdateUser(ctx context.Context, user entity.User) error {
	if m.UpdateUserFunc != nil {
		return m.UpdateUserFunc(ctx, user)
	}
	return nil
}
func (m *mockUserUC) UpdateLastOnline(ctx context.Context, id string) error {
	if m.UpdateLastOnlineFunc != nil {
		return m.UpdateLastOnlineFunc(ctx, id)
	}
	return nil
}
func (m *mockUserUC) DeleteUser(ctx context.Context, id string) error {
	if m.DeleteUserFunc != nil {
		return m.DeleteUserFunc(ctx, id)
	}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newUserRouter(uc *mockUserUC) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handlers.NewUserHandler(uc)

	// Inject userID for protected routes
	withUser := func(c *gin.Context) {
		c.Set("userID", "uid1")
		c.Next()
	}

	r.GET("/v1/auth/me", withUser, h.GetCurrentUser)
	r.GET("/v1/users", withUser, h.ListUsers)
	r.POST("/v1/users", withUser, h.CreateUser)
	r.GET("/v1/users/:id", withUser, h.GetUserByID)
	r.PUT("/v1/users/:id", withUser, h.UpdateUser)
	r.DELETE("/v1/users/:id", withUser, h.DeleteUser)
	return r
}

// ─── GetCurrentUser ───────────────────────────────────────────────────────────

func TestUserHandler_GetCurrentUser_Success(t *testing.T) {
	uc := &mockUserUC{
		GetByIDFunc: func(ctx context.Context, id string) (entity.User, error) {
			return entity.User{ID: "uid1", Email: "a@b.com"}, nil
		},
	}
	r := newUserRouter(uc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "a@b.com", resp["email"])
}

func TestUserHandler_GetCurrentUser_NotFound(t *testing.T) {
	r := newUserRouter(&mockUserUC{}) // GetByID returns ErrUserNotFound by default
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── ListUsers ────────────────────────────────────────────────────────────────

func TestUserHandler_ListUsers(t *testing.T) {
	uc := &mockUserUC{
		ListUsersFunc: func(ctx context.Context) ([]entity.User, error) {
			return []entity.User{{ID: "1", Email: "a@b.com"}, {ID: "2", Email: "c@d.com"}}, nil
		},
	}
	r := newUserRouter(uc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(2), resp["count"])
}

// ─── CreateUser ───────────────────────────────────────────────────────────────

func TestUserHandler_CreateUser_BadBody(t *testing.T) {
	r := newUserRouter(&mockUserUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/users", jsonBody(map[string]string{"email": "bad"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_CreateUser_Success(t *testing.T) {
	uc := &mockUserUC{
		CreateUserFunc: func(ctx context.Context, user entity.User) (string, error) {
			return "new-uid", nil
		},
	}
	r := newUserRouter(uc)
	w := httptest.NewRecorder()
	body := map[string]interface{}{
		"email":    "new@b.com",
		"password": "pass123",
		"roles":    []string{"admin"},
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/users", jsonBody(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "new-uid", resp["id"])
}

// ─── GetUserByID ──────────────────────────────────────────────────────────────

func TestUserHandler_GetUserByID_Success(t *testing.T) {
	uc := &mockUserUC{
		GetByIDFunc: func(ctx context.Context, id string) (entity.User, error) {
			return entity.User{ID: id, Email: "x@y.com"}, nil
		},
	}
	r := newUserRouter(uc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/users/uid42", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_GetUserByID_NotFound(t *testing.T) {
	r := newUserRouter(&mockUserUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/users/missing", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── DeleteUser ───────────────────────────────────────────────────────────────

func TestUserHandler_DeleteUser_Success(t *testing.T) {
	r := newUserRouter(&mockUserUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/v1/users/uid1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
