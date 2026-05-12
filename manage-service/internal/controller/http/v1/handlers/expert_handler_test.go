package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"manage-service/internal/controller/http/v1/handlers"
	"manage-service/internal/domain"
	"manage-service/internal/domain/entity"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── Mock Expert UseCase ──────────────────────────────────────────────────────

type mockExpertHandlerUC struct {
	SaveEvaluationFunc              func(ctx context.Context, applicantID int64, expertID, userID, userName, role string, scores []entity.ExpertEvaluation, complete bool) error
	ListEvaluationsFunc             func(ctx context.Context, applicantID int64, currentUserID, role string) ([]entity.ExpertEvaluation, error)
	GetEvaluationCriteriaFunc       func(ctx context.Context) ([]entity.EvaluationCriteria, error)
	GetEvaluationCriteriaForAppFunc func(ctx context.Context, applicantID int64) ([]entity.EvaluationCriteria, string, error)
	GetExpertSlotsFunc              func(ctx context.Context, programID int64) ([]entity.ExpertSlot, error)
	ListExpertsFunc                 func(ctx context.Context) ([]entity.User, error)
}

func (m *mockExpertHandlerUC) SaveExpertEvaluation(ctx context.Context, applicantID int64, expertID, userID, userName, role string, scores []entity.ExpertEvaluation, complete bool) error {
	if m.SaveEvaluationFunc != nil {
		return m.SaveEvaluationFunc(ctx, applicantID, expertID, userID, userName, role, scores, complete)
	}
	return nil
}
func (m *mockExpertHandlerUC) ListExpertEvaluations(ctx context.Context, applicantID int64, currentUserID, role string) ([]entity.ExpertEvaluation, error) {
	if m.ListEvaluationsFunc != nil {
		return m.ListEvaluationsFunc(ctx, applicantID, currentUserID, role)
	}
	return []entity.ExpertEvaluation{}, nil
}
func (m *mockExpertHandlerUC) GetEvaluationCriteria(ctx context.Context) ([]entity.EvaluationCriteria, error) {
	if m.GetEvaluationCriteriaFunc != nil {
		return m.GetEvaluationCriteriaFunc(ctx)
	}
	return []entity.EvaluationCriteria{{Code: "C1", Title: "Критерий 1", MaxScore: 10}}, nil
}
func (m *mockExpertHandlerUC) GetEvaluationCriteriaForApplicant(ctx context.Context, applicantID int64) ([]entity.EvaluationCriteria, string, error) {
	if m.GetEvaluationCriteriaForAppFunc != nil {
		return m.GetEvaluationCriteriaForAppFunc(ctx, applicantID)
	}
	return []entity.EvaluationCriteria{{Code: "C1"}}, "default", nil
}
func (m *mockExpertHandlerUC) GetApplicantScoringScheme(ctx context.Context, applicantID int64) (string, error) {
	return "default", nil
}
func (m *mockExpertHandlerUC) SetApplicantScoringScheme(ctx context.Context, applicantID int64, scheme string, role string) error {
	return nil
}
func (m *mockExpertHandlerUC) CreateCriteria(ctx context.Context, c entity.EvaluationCriteria) error {
	return nil
}
func (m *mockExpertHandlerUC) UpdateCriteria(ctx context.Context, c entity.EvaluationCriteria) error {
	return nil
}
func (m *mockExpertHandlerUC) DeleteCriteria(ctx context.Context, code string) error { return nil }
func (m *mockExpertHandlerUC) GetExpertSlots(ctx context.Context, programID int64) ([]entity.ExpertSlot, error) {
	if m.GetExpertSlotsFunc != nil {
		return m.GetExpertSlotsFunc(ctx, programID)
	}
	return []entity.ExpertSlot{}, nil
}
func (m *mockExpertHandlerUC) AssignExpertSlot(ctx context.Context, userID string, slotNumber int, programID int64, requesterRole string) error {
	return nil
}
func (m *mockExpertHandlerUC) ListExperts(ctx context.Context) ([]entity.User, error) {
	if m.ListExpertsFunc != nil {
		return m.ListExpertsFunc(ctx)
	}
	return []entity.User{}, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newExpertRouter(uc *mockExpertHandlerUC) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handlers.NewExpertHandler(uc)
	r.GET("/v1/criteria", h.ListCriteria)
	r.POST("/v1/criteria", h.CreateCriteria)
	r.PUT("/v1/criteria/:code", h.UpdateCriteria)
	r.DELETE("/v1/criteria/:code", h.DeleteCriteria)
	r.GET("/v1/applicants/:id/criteria", h.GetEvaluationCriteria)
	r.GET("/v1/applicants/:id/evaluations", h.ListExpertEvaluations)
	r.PUT("/v1/applicants/:id/evaluations", h.SaveExpertEvaluation)
	r.GET("/v1/experts/slots", h.GetExpertSlots)
	r.POST("/v1/experts/slots", h.AssignExpertSlot)
	r.GET("/v1/experts", h.ListExperts)
	return r
}

// ─── ListCriteria ─────────────────────────────────────────────────────────────

func TestExpertHandler_ListCriteria_Success(t *testing.T) {
	r := newExpertRouter(&mockExpertHandlerUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/criteria", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp []entity.EvaluationCriteria
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Len(t, resp, 1)
}

// ─── CreateCriteria ───────────────────────────────────────────────────────────

func TestExpertHandler_CreateCriteria_MissingFields(t *testing.T) {
	r := newExpertRouter(&mockExpertHandlerUC{})
	w := httptest.NewRecorder()
	// Missing code/title/max_score
	req := httptest.NewRequest(http.MethodPost, "/v1/criteria", jsonBodyM(map[string]interface{}{"code": "C1"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExpertHandler_CreateCriteria_Success(t *testing.T) {
	r := newExpertRouter(&mockExpertHandlerUC{})
	w := httptest.NewRecorder()
	body := map[string]interface{}{"code": "C1", "title": "Критерий 1", "max_score": 10}
	req := httptest.NewRequest(http.MethodPost, "/v1/criteria", jsonBodyM(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ─── GetEvaluationCriteria ────────────────────────────────────────────────────

func TestExpertHandler_GetEvaluationCriteria_InvalidID(t *testing.T) {
	r := newExpertRouter(&mockExpertHandlerUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/applicants/abc/criteria", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExpertHandler_GetEvaluationCriteria_Success(t *testing.T) {
	r := newExpertRouter(&mockExpertHandlerUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/applicants/1/criteria", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ─── ListExpertEvaluations ────────────────────────────────────────────────────

func TestExpertHandler_ListExpertEvaluations_Success(t *testing.T) {
	uc := &mockExpertHandlerUC{
		ListEvaluationsFunc: func(ctx context.Context, applicantID int64, currentUserID, role string) ([]entity.ExpertEvaluation, error) {
			return []entity.ExpertEvaluation{{ExpertID: "e1", Score: 7}}, nil
		},
	}
	r := newExpertRouter(uc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/applicants/1/evaluations?user_id=e1&user_role=expert", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp []entity.ExpertEvaluation
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Len(t, resp, 1)
}

// ─── SaveExpertEvaluation ─────────────────────────────────────────────────────

func TestExpertHandler_SaveExpertEvaluation_BadBody(t *testing.T) {
	r := newExpertRouter(&mockExpertHandlerUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/applicants/1/evaluations", jsonBodyM("not-object"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExpertHandler_SaveExpertEvaluation_Immutable(t *testing.T) {
	uc := &mockExpertHandlerUC{
		SaveEvaluationFunc: func(ctx context.Context, applicantID int64, expertID, userID, userName, role string, scores []entity.ExpertEvaluation, complete bool) error {
			return domain.ErrEvaluationImmutable
		},
	}
	r := newExpertRouter(uc)
	w := httptest.NewRecorder()
	body := map[string]interface{}{
		"expert_id": "e1",
		"user_id":   "e1",
		"user_name": "Эксперт",
		"user_role": "expert",
		"scores":    []map[string]interface{}{{"category": "C1", "score": 5}},
	}
	req := httptest.NewRequest(http.MethodPut, "/v1/applicants/1/evaluations", jsonBodyM(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestExpertHandler_SaveExpertEvaluation_ScoreExceedsMax(t *testing.T) {
	uc := &mockExpertHandlerUC{
		SaveEvaluationFunc: func(ctx context.Context, applicantID int64, expertID, userID, userName, role string, scores []entity.ExpertEvaluation, complete bool) error {
			return domain.ErrScoreExceedsMax
		},
	}
	r := newExpertRouter(uc)
	w := httptest.NewRecorder()
	body := map[string]interface{}{
		"expert_id": "e1",
		"user_id":   "e1",
		"scores":    []map[string]interface{}{{"category": "C1", "score": 999}},
	}
	req := httptest.NewRequest(http.MethodPut, "/v1/applicants/1/evaluations", jsonBodyM(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExpertHandler_SaveExpertEvaluation_Success(t *testing.T) {
	r := newExpertRouter(&mockExpertHandlerUC{})
	w := httptest.NewRecorder()
	body := map[string]interface{}{
		"expert_id": "e1",
		"user_id":   "e1",
		"user_name": "Эксперт",
		"user_role": "expert",
		"scores":    []map[string]interface{}{{"category": "C1", "score": 7}},
	}
	req := httptest.NewRequest(http.MethodPut, "/v1/applicants/1/evaluations", jsonBodyM(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ─── ListExperts ──────────────────────────────────────────────────────────────

func TestExpertHandler_ListExperts_Success(t *testing.T) {
	uc := &mockExpertHandlerUC{
		ListExpertsFunc: func(ctx context.Context) ([]entity.User, error) {
			return []entity.User{{ID: "e1"}, {ID: "e2"}}, nil
		},
	}
	r := newExpertRouter(uc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/experts", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp []entity.User
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Len(t, resp, 2)
}
