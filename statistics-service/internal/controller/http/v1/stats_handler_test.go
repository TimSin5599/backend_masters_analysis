package v1_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	v1 "statistics-service/internal/controller/http/v1"
	"statistics-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── Mock StatsRepo ───────────────────────────────────────────────────────────

type mockStatsRepo struct {
	GetGlobalStatsFunc func(ctx context.Context, programID int64) (usecase.GlobalStats, error)
	GetDynamicsFunc    func(ctx context.Context, period string, programID int64) ([]usecase.DailyStats, error)
}

func (m *mockStatsRepo) GetGlobalStats(ctx context.Context, programID int64) (usecase.GlobalStats, error) {
	if m.GetGlobalStatsFunc != nil {
		return m.GetGlobalStatsFunc(ctx, programID)
	}
	return usecase.GlobalStats{TotalApplicants: 42}, nil
}

func (m *mockStatsRepo) GetDynamics(ctx context.Context, period string, programID int64) ([]usecase.DailyStats, error) {
	if m.GetDynamicsFunc != nil {
		return m.GetDynamicsFunc(ctx, period, programID)
	}
	return []usecase.DailyStats{{Date: "2024-04-23", Submitted: 5}}, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newStatsRouter(repo *mockStatsRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	uc := usecase.New(repo)
	v1.NewRouter(r, uc, "")
	return r
}

// ─── GET /v1/stats/overview ───────────────────────────────────────────────────

func TestStatsHandler_Overview_Success(t *testing.T) {
	r := newStatsRouter(&mockStatsRepo{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/stats/overview", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp usecase.GlobalStats
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, int64(42), resp.TotalApplicants)
}

func TestStatsHandler_Overview_WithProgramID(t *testing.T) {
	var capturedID int64
	repo := &mockStatsRepo{
		GetGlobalStatsFunc: func(ctx context.Context, programID int64) (usecase.GlobalStats, error) {
			capturedID = programID
			return usecase.GlobalStats{}, nil
		},
	}
	r := newStatsRouter(repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/stats/overview?program_id=7", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, int64(7), capturedID)
}

func TestStatsHandler_Overview_Error(t *testing.T) {
	repo := &mockStatsRepo{
		GetGlobalStatsFunc: func(ctx context.Context, programID int64) (usecase.GlobalStats, error) {
			return usecase.GlobalStats{}, errors.New("db error")
		},
	}
	r := newStatsRouter(repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/stats/overview", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ─── GET /v1/stats/dynamics ───────────────────────────────────────────────────

func TestStatsHandler_Dynamics_Success(t *testing.T) {
	r := newStatsRouter(&mockStatsRepo{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/stats/dynamics", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp []usecase.DailyStats
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Len(t, resp, 1)
	assert.Equal(t, "2024-04-23", resp[0].Date)
}

func TestStatsHandler_Dynamics_WeeklyPeriod(t *testing.T) {
	var capturedPeriod string
	repo := &mockStatsRepo{
		GetDynamicsFunc: func(ctx context.Context, period string, programID int64) ([]usecase.DailyStats, error) {
			capturedPeriod = period
			return nil, nil
		},
	}
	r := newStatsRouter(repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/stats/dynamics?period=weekly", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "weekly", capturedPeriod)
}

func TestStatsHandler_Dynamics_InvalidPeriodFallsBackToDaily(t *testing.T) {
	var capturedPeriod string
	repo := &mockStatsRepo{
		GetDynamicsFunc: func(ctx context.Context, period string, programID int64) ([]usecase.DailyStats, error) {
			capturedPeriod = period
			return nil, nil
		},
	}
	r := newStatsRouter(repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/stats/dynamics?period=hourly", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "daily", capturedPeriod)
}

func TestStatsHandler_Dynamics_Error(t *testing.T) {
	repo := &mockStatsRepo{
		GetDynamicsFunc: func(ctx context.Context, period string, programID int64) ([]usecase.DailyStats, error) {
			return nil, errors.New("db error")
		},
	}
	r := newStatsRouter(repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/stats/dynamics", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
