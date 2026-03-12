package usecase

import (
	"context"
)

type StatsUseCase struct {
	repo StatsRepo
}

type GlobalStats struct {
	TotalApplicants int64            `json:"total_applicants"`
	ByStatus        map[string]int64 `json:"by_status"`
	ProcessedToday  int64            `json:"processed_today"`
}

type DailyStats struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type StatsRepo interface {
	GetGlobalStats(ctx context.Context) (GlobalStats, error)
	GetDailyProcessingDynamics(ctx context.Context) ([]DailyStats, error)
}

func New(r StatsRepo) *StatsUseCase {
	return &StatsUseCase{repo: r}
}

func (uc *StatsUseCase) GetOverview(ctx context.Context) (GlobalStats, error) {
	return uc.repo.GetGlobalStats(ctx)
}

func (uc *StatsUseCase) GetDynamics(ctx context.Context) ([]DailyStats, error) {
	return uc.repo.GetDailyProcessingDynamics(ctx)
}
