package usecase

import "context"

type StatsUseCase struct {
	repo StatsRepo
}

// GlobalStats содержит сводные метрики по набору.
type GlobalStats struct {
	TotalApplicants    int64           `json:"total_applicants"     example:"128"`
	AIProcessing       int64           `json:"ai_processing"        example:"5"`
	Verifying          int64           `json:"verifying"            example:"20"`
	Assessing          int64           `json:"assessing"            example:"15"`
	Evaluated          int64           `json:"evaluated"            example:"60"`
	AvgScore           float64         `json:"avg_score"            example:"78.5"`
	MaxScore           float64         `json:"max_score"            example:"95.0"`
	DocProcessingByDay []DayProcessing `json:"doc_processing_by_day"`
	AIErrorsByCategory []CategoryError `json:"ai_errors_by_category"`
}

// DayProcessing — среднее время обработки документов за один день.
type DayProcessing struct {
	Date       string  `json:"date"        example:"2024-04-22"`
	AvgMinutes float64 `json:"avg_minutes" example:"12.5"`
}

// CategoryError — количество ошибок ИИ по типу документа.
type CategoryError struct {
	Category string `json:"category" example:"diploma"`
	Count    int64  `json:"count"    example:"3"`
}

// DailyStats — срез количества абитуриентов по статусам за один период.
type DailyStats struct {
	Date      string `json:"date"      example:"2024-04-22"`
	Submitted int64  `json:"submitted" example:"5"`
	Verifying int64  `json:"verifying" example:"3"`
	Assessing int64  `json:"assessing" example:"2"`
	Evaluated int64  `json:"evaluated" example:"1"`
}

type StatsRepo interface {
	GetGlobalStats(ctx context.Context, programID int64) (GlobalStats, error)
	GetDynamics(ctx context.Context, period string, programID int64) ([]DailyStats, error)
}

func New(r StatsRepo) *StatsUseCase {
	return &StatsUseCase{repo: r}
}

func (uc *StatsUseCase) GetOverview(ctx context.Context, programID int64) (GlobalStats, error) {
	return uc.repo.GetGlobalStats(ctx, programID)
}

func (uc *StatsUseCase) GetDynamics(ctx context.Context, period string, programID int64) ([]DailyStats, error) {
	switch period {
	case "weekly", "monthly":
	default:
		period = "daily"
	}
	return uc.repo.GetDynamics(ctx, period, programID)
}
