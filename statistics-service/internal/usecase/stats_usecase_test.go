package usecase_test

import (
	"context"
	"testing"

	"statistics-service/internal/usecase"

	"github.com/stretchr/testify/assert"
)

type mockStatsRepo struct {
	GetGlobalStatsFunc func(ctx context.Context, programID int64) (usecase.GlobalStats, error)
	GetDynamicsFunc    func(ctx context.Context, period string, programID int64) ([]usecase.DailyStats, error)
}

func (m *mockStatsRepo) GetGlobalStats(ctx context.Context, programID int64) (usecase.GlobalStats, error) {
	return m.GetGlobalStatsFunc(ctx, programID)
}

func (m *mockStatsRepo) GetDynamics(ctx context.Context, period string, programID int64) ([]usecase.DailyStats, error) {
	return m.GetDynamicsFunc(ctx, period, programID)
}

func TestStatsUseCase_GetOverview(t *testing.T) {
	repo := &mockStatsRepo{
		GetGlobalStatsFunc: func(ctx context.Context, programID int64) (usecase.GlobalStats, error) {
			return usecase.GlobalStats{TotalApplicants: 10}, nil
		},
	}

	uc := usecase.New(repo)
	stats, err := uc.GetOverview(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, int64(10), stats.TotalApplicants)
}

func TestStatsUseCase_GetDynamics(t *testing.T) {
	repo := &mockStatsRepo{
		GetDynamicsFunc: func(ctx context.Context, period string, programID int64) ([]usecase.DailyStats, error) {
			return []usecase.DailyStats{{Date: "2024-04-23"}}, nil
		},
	}

	uc := usecase.New(repo)

	t.Run("Valid Period Weekly", func(t *testing.T) {
		repo.GetDynamicsFunc = func(ctx context.Context, period string, programID int64) ([]usecase.DailyStats, error) {
			assert.Equal(t, "weekly", period)
			return nil, nil
		}
		_, err := uc.GetDynamics(context.Background(), "weekly", 1)
		assert.NoError(t, err)
	})

	t.Run("Invalid Period Falls Back To Daily", func(t *testing.T) {
		repo.GetDynamicsFunc = func(ctx context.Context, period string, programID int64) ([]usecase.DailyStats, error) {
			assert.Equal(t, "daily", period)
			return nil, nil
		}
		_, err := uc.GetDynamics(context.Background(), "invalid_period", 1)
		assert.NoError(t, err)
	})
}
