package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"manage-service/internal/controller/http/v1/handlers"
	"manage-service/internal/domain/entity"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── Mock Applicant UseCase ───────────────────────────────────────────────────

type mockApplicantUC struct {
	CreateApplicantFunc      func(ctx context.Context, programID int64, firstName, lastName, patronymic string) (entity.Applicant, error)
	ListApplicantsFunc       func(ctx context.Context, programID int64) ([]entity.Applicant, error)
	DeleteApplicantFunc      func(ctx context.Context, id int64) error
	GetApplicantDataFunc     func(ctx context.Context, applicantID int64, category string) (interface{}, error)
	UpdateApplicantDataFunc  func(ctx context.Context, applicantID int64, category string, rawData map[string]interface{}) error
	DeleteApplicantDataFunc  func(ctx context.Context, applicantID int64, category string, dataID int64) error
	TransferToOperatorFunc   func(ctx context.Context, applicantID int64) error
	TransferToExpertsFunc    func(ctx context.Context, applicantID int64, confirmedBy string) error
}

func (m *mockApplicantUC) CreateApplicant(ctx context.Context, programID int64, firstName, lastName, patronymic string) (entity.Applicant, error) {
	if m.CreateApplicantFunc != nil {
		return m.CreateApplicantFunc(ctx, programID, firstName, lastName, patronymic)
	}
	return entity.Applicant{ID: 1, ProgramID: programID}, nil
}
func (m *mockApplicantUC) ListApplicants(ctx context.Context, programID int64) ([]entity.Applicant, error) {
	if m.ListApplicantsFunc != nil {
		return m.ListApplicantsFunc(ctx, programID)
	}
	return []entity.Applicant{}, nil
}
func (m *mockApplicantUC) DeleteApplicant(ctx context.Context, id int64) error {
	if m.DeleteApplicantFunc != nil {
		return m.DeleteApplicantFunc(ctx, id)
	}
	return nil
}
func (m *mockApplicantUC) GetApplicantData(ctx context.Context, applicantID int64, category string) (interface{}, error) {
	if m.GetApplicantDataFunc != nil {
		return m.GetApplicantDataFunc(ctx, applicantID, category)
	}
	return map[string]string{"key": "value"}, nil
}
func (m *mockApplicantUC) UpdateApplicantData(ctx context.Context, applicantID int64, category string, rawData map[string]interface{}) error {
	if m.UpdateApplicantDataFunc != nil {
		return m.UpdateApplicantDataFunc(ctx, applicantID, category, rawData)
	}
	return nil
}
func (m *mockApplicantUC) DeleteApplicantData(ctx context.Context, applicantID int64, category string, dataID int64) error {
	if m.DeleteApplicantDataFunc != nil {
		return m.DeleteApplicantDataFunc(ctx, applicantID, category, dataID)
	}
	return nil
}
func (m *mockApplicantUC) TransferToOperator(ctx context.Context, applicantID int64) error {
	if m.TransferToOperatorFunc != nil {
		return m.TransferToOperatorFunc(ctx, applicantID)
	}
	return nil
}
func (m *mockApplicantUC) TransferToExperts(ctx context.Context, applicantID int64, confirmedBy string) error {
	if m.TransferToExpertsFunc != nil {
		return m.TransferToExpertsFunc(ctx, applicantID, confirmedBy)
	}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newApplicantRouter(uc *mockApplicantUC) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handlers.NewApplicantHandler(uc)
	r.GET("/v1/applicants", h.ListApplicants)
	r.POST("/v1/applicants", h.CreateApplicant)
	r.DELETE("/v1/applicants/:id", h.DeleteApplicant)
	r.GET("/v1/applicants/:id/data", h.GetApplicantData)
	r.PATCH("/v1/applicants/:id/data", h.UpdateApplicantData)
	r.DELETE("/v1/applicants/:id/data/:category/:dataId", h.DeleteApplicantData)
	r.POST("/v1/applicants/:id/transfer-to-operator", h.TransferToOperator)
	r.POST("/v1/applicants/:id/transfer-to-experts", h.TransferToExperts)
	return r
}

func jsonBodyM(v interface{}) *bytes.Buffer {
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

// ─── ListApplicants ───────────────────────────────────────────────────────────

func TestApplicantHandler_ListApplicants_Success(t *testing.T) {
	uc := &mockApplicantUC{
		ListApplicantsFunc: func(ctx context.Context, programID int64) ([]entity.Applicant, error) {
			return []entity.Applicant{{ID: 1}, {ID: 2}}, nil
		},
	}
	r := newApplicantRouter(uc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/applicants", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp []entity.Applicant
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Len(t, resp, 2)
}

func TestApplicantHandler_ListApplicants_WithProgramID(t *testing.T) {
	var capturedProgramID int64
	uc := &mockApplicantUC{
		ListApplicantsFunc: func(ctx context.Context, programID int64) ([]entity.Applicant, error) {
			capturedProgramID = programID
			return []entity.Applicant{}, nil
		},
	}
	r := newApplicantRouter(uc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/applicants?program_id=5", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, int64(5), capturedProgramID)
}

// ─── CreateApplicant ──────────────────────────────────────────────────────────

func TestApplicantHandler_CreateApplicant_Success(t *testing.T) {
	uc := &mockApplicantUC{
		CreateApplicantFunc: func(ctx context.Context, programID int64, firstName, lastName, patronymic string) (entity.Applicant, error) {
			return entity.Applicant{ID: 42, ProgramID: programID}, nil
		},
	}
	r := newApplicantRouter(uc)
	w := httptest.NewRecorder()
	body := map[string]interface{}{"program_id": 1, "first_name": "Иван", "last_name": "Иванов", "patronymic": "Иванович"}
	req := httptest.NewRequest(http.MethodPost, "/v1/applicants", jsonBodyM(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp entity.Applicant
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, int64(42), resp.ID)
}

func TestApplicantHandler_CreateApplicant_BadBody(t *testing.T) {
	r := newApplicantRouter(&mockApplicantUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/applicants", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── DeleteApplicant ──────────────────────────────────────────────────────────

func TestApplicantHandler_DeleteApplicant_InvalidID(t *testing.T) {
	r := newApplicantRouter(&mockApplicantUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/v1/applicants/abc", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestApplicantHandler_DeleteApplicant_Success(t *testing.T) {
	r := newApplicantRouter(&mockApplicantUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/v1/applicants/1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestApplicantHandler_DeleteApplicant_Error(t *testing.T) {
	uc := &mockApplicantUC{
		DeleteApplicantFunc: func(ctx context.Context, id int64) error {
			return errors.New("db error")
		},
	}
	r := newApplicantRouter(uc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/v1/applicants/1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ─── GetApplicantData ─────────────────────────────────────────────────────────

func TestApplicantHandler_GetApplicantData_InvalidID(t *testing.T) {
	r := newApplicantRouter(&mockApplicantUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/applicants/abc/data", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestApplicantHandler_GetApplicantData_Success(t *testing.T) {
	r := newApplicantRouter(&mockApplicantUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/applicants/1/data?category=passport", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ─── TransferToOperator ───────────────────────────────────────────────────────

func TestApplicantHandler_TransferToOperator_Success(t *testing.T) {
	r := newApplicantRouter(&mockApplicantUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/applicants/1/transfer-to-operator", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ─── TransferToExperts ────────────────────────────────────────────────────────

func TestApplicantHandler_TransferToExperts_Success(t *testing.T) {
	r := newApplicantRouter(&mockApplicantUC{})
	w := httptest.NewRecorder()
	body := map[string]string{"user_name": "Иван Иванов", "user_role": "manager"}
	req := httptest.NewRequest(http.MethodPost, "/v1/applicants/1/transfer-to-experts", jsonBodyM(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
