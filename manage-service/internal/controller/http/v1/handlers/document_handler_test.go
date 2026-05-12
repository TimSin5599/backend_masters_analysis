package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"manage-service/internal/controller/http/v1/handlers"
	"manage-service/internal/domain/entity"
)

// ─── Mock Document UseCase ────────────────────────────────────────────────────

type mockDocumentHandlerUC struct {
	GetDocumentStatusFunc       func(ctx context.Context, documentID int64) (string, error)
	UpdateDocumentStatusFunc    func(ctx context.Context, documentID int64, status string) error
	GetDocumentsFunc            func(ctx context.Context, applicantID int64) ([]entity.Document, error)
	ReprocessLatestDocumentFunc func(ctx context.Context, applicantID int64, category string) (int64, error)
	ReprocessDocumentFunc       func(ctx context.Context, documentID int64) error
}

func (m *mockDocumentHandlerUC) UploadDocument(ctx context.Context, applicantID int64, category string, fileName string, content []byte, docType string) (entity.Document, error) {
	return entity.Document{ID: 1}, nil
}
func (m *mockDocumentHandlerUC) GetDocuments(ctx context.Context, applicantID int64) ([]entity.Document, error) {
	if m.GetDocumentsFunc != nil {
		return m.GetDocumentsFunc(ctx, applicantID)
	}
	return []entity.Document{}, nil
}
func (m *mockDocumentHandlerUC) ViewDocument(ctx context.Context, applicantID int64, category string) ([]byte, string, string, error) {
	return nil, "", "", nil
}
func (m *mockDocumentHandlerUC) ViewDocumentByID(ctx context.Context, documentID int64) ([]byte, string, string, error) {
	return []byte("pdf"), "application/pdf", "doc.pdf", nil
}
func (m *mockDocumentHandlerUC) DeleteDocument(ctx context.Context, applicantID int64, documentID int64) error {
	return nil
}
func (m *mockDocumentHandlerUC) ReprocessLatestDocument(ctx context.Context, applicantID int64, category string) (int64, error) {
	if m.ReprocessLatestDocumentFunc != nil {
		return m.ReprocessLatestDocumentFunc(ctx, applicantID, category)
	}
	return 10, nil
}
func (m *mockDocumentHandlerUC) ReprocessDocument(ctx context.Context, documentID int64) error {
	if m.ReprocessDocumentFunc != nil {
		return m.ReprocessDocumentFunc(ctx, documentID)
	}
	return nil
}
func (m *mockDocumentHandlerUC) ChangeDocumentCategory(ctx context.Context, documentID int64, newCategory string) error {
	return nil
}
func (m *mockDocumentHandlerUC) GetDocumentStatus(ctx context.Context, documentID int64) (string, error) {
	if m.GetDocumentStatusFunc != nil {
		return m.GetDocumentStatusFunc(ctx, documentID)
	}
	return "processing", nil
}
func (m *mockDocumentHandlerUC) UpdateDocumentStatus(ctx context.Context, documentID int64, status string) error {
	if m.UpdateDocumentStatusFunc != nil {
		return m.UpdateDocumentStatusFunc(ctx, documentID, status)
	}
	return nil
}
func (m *mockDocumentHandlerUC) GetQueueTasks(ctx context.Context, applicantID int64) ([]entity.DocumentQueueTask, error) {
	return []entity.DocumentQueueTask{}, nil
}
func (m *mockDocumentHandlerUC) ProcessAIResult(ctx context.Context, applicantID int64, documentID int64, taskCategory string, rawData map[string]string) error {
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newDocumentRouter(uc *mockDocumentHandlerUC) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handlers.NewDocumentHandler(uc)
	r.GET("/v1/documents/:id/status", h.GetDocumentStatus)
	r.PATCH("/v1/documents/:id/status", h.PatchDocumentStatus)
	r.GET("/v1/applicants/:id/documents", h.ListDocuments)
	r.POST("/v1/applicants/:id/documents/reprocess", h.ReprocessLatestDocument)
	r.POST("/v1/documents/:id/reprocess", h.ReprocessDocument)
	r.GET("/v1/applicants/:id/queue-status", h.GetQueueStatus)
	r.PATCH("/v1/documents/:id/category", h.ChangeDocumentCategory)
	return r
}

// ─── GetDocumentStatus ────────────────────────────────────────────────────────

func TestDocumentHandler_GetDocumentStatus_InvalidID(t *testing.T) {
	r := newDocumentRouter(&mockDocumentHandlerUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/documents/abc/status", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDocumentHandler_GetDocumentStatus_NotFound(t *testing.T) {
	uc := &mockDocumentHandlerUC{
		GetDocumentStatusFunc: func(ctx context.Context, documentID int64) (string, error) {
			return "", errors.New("not found")
		},
	}
	r := newDocumentRouter(uc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/documents/99/status", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDocumentHandler_GetDocumentStatus_Success(t *testing.T) {
	r := newDocumentRouter(&mockDocumentHandlerUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/documents/1/status", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "processing", resp["status"])
}

// ─── PatchDocumentStatus ──────────────────────────────────────────────────────

func TestDocumentHandler_PatchDocumentStatus_InvalidID(t *testing.T) {
	r := newDocumentRouter(&mockDocumentHandlerUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/v1/documents/abc/status", jsonBodyM(map[string]string{"status": "completed"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDocumentHandler_PatchDocumentStatus_DisallowedStatus(t *testing.T) {
	r := newDocumentRouter(&mockDocumentHandlerUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/v1/documents/1/status", jsonBodyM(map[string]string{"status": "failed"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDocumentHandler_PatchDocumentStatus_EmptyStatus(t *testing.T) {
	r := newDocumentRouter(&mockDocumentHandlerUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/v1/documents/1/status", jsonBodyM(map[string]string{}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDocumentHandler_PatchDocumentStatus_Success(t *testing.T) {
	r := newDocumentRouter(&mockDocumentHandlerUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/v1/documents/1/status", jsonBodyM(map[string]string{"status": "completed"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ─── ListDocuments ────────────────────────────────────────────────────────────

func TestDocumentHandler_ListDocuments_InvalidID(t *testing.T) {
	r := newDocumentRouter(&mockDocumentHandlerUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/applicants/abc/documents", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDocumentHandler_ListDocuments_Success(t *testing.T) {
	uc := &mockDocumentHandlerUC{
		GetDocumentsFunc: func(ctx context.Context, applicantID int64) ([]entity.Document, error) {
			return []entity.Document{{ID: 1, ApplicantID: applicantID}}, nil
		},
	}
	r := newDocumentRouter(uc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/applicants/1/documents", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ─── ReprocessLatestDocument ──────────────────────────────────────────────────

func TestDocumentHandler_ReprocessLatestDocument_NoCategory(t *testing.T) {
	r := newDocumentRouter(&mockDocumentHandlerUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/applicants/1/documents/reprocess", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDocumentHandler_ReprocessLatestDocument_Success(t *testing.T) {
	r := newDocumentRouter(&mockDocumentHandlerUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/applicants/1/documents/reprocess?category=passport", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(10), resp["document_id"])
}

// ─── ChangeDocumentCategory ───────────────────────────────────────────────────

func TestDocumentHandler_ChangeDocumentCategory_NoCategory(t *testing.T) {
	r := newDocumentRouter(&mockDocumentHandlerUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/v1/documents/1/category", jsonBodyM(map[string]string{}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDocumentHandler_ChangeDocumentCategory_Success(t *testing.T) {
	r := newDocumentRouter(&mockDocumentHandlerUC{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/v1/documents/1/category", jsonBodyM(map[string]string{"category": "diploma"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
