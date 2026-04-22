package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"statistics-service/internal/usecase"
)

type StatsRepo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *StatsRepo {
	return &StatsRepo{pool: pool}
}

func (r *StatsRepo) GetGlobalStats(ctx context.Context, programID int64) (usecase.GlobalStats, error) {
	var s usecase.GlobalStats

	// Applicant counts + score stats
	row := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'verifying'),
			COUNT(*) FILTER (WHERE status = 'assessed'),
			COUNT(*) FILTER (WHERE status = 'completed'),
			COALESCE(AVG(aggregated_score) FILTER (WHERE status = 'completed'), 0),
			COALESCE(MAX(aggregated_score) FILTER (WHERE status = 'completed'), 0)
		FROM applicants
		WHERE ($1 = 0 OR program_id = $1)
	`, programID)
	if err := row.Scan(
		&s.TotalApplicants, &s.Verifying, &s.Assessing, &s.Evaluated,
		&s.AvgScore, &s.MaxScore,
	); err != nil {
		return s, fmt.Errorf("GetGlobalStats applicant counts: %w", err)
	}

	// AI processing: documents currently being processed by the AI pipeline
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM applicants_document d
		JOIN applicants a ON d.applicant_id = a.id
		WHERE d.status IN ('pending', 'classifying', 'classified', 'extracting')
		  AND ($1 = 0 OR a.program_id = $1)
	`, programID).Scan(&s.AIProcessing); err != nil {
		return s, fmt.Errorf("GetGlobalStats ai_processing: %w", err)
	}

	// AI errors by document category
	errRows, err := r.pool.Query(ctx, `
		SELECT COALESCE(d.file_type, 'unknown'), COUNT(*)
		FROM applicants_document d
		JOIN applicants a ON d.applicant_id = a.id
		WHERE d.status IN ('classification_failed', 'extraction_failed')
		  AND ($1 = 0 OR a.program_id = $1)
		GROUP BY d.file_type
		ORDER BY COUNT(*) DESC
	`, programID)
	if err != nil {
		return s, fmt.Errorf("GetGlobalStats ai_errors: %w", err)
	}
	defer errRows.Close()

	s.AIErrorsByCategory = []usecase.CategoryError{}
	for errRows.Next() {
		var ce usecase.CategoryError
		if err := errRows.Scan(&ce.Category, &ce.Count); err != nil {
			return s, err
		}
		s.AIErrorsByCategory = append(s.AIErrorsByCategory, ce)
	}

	// Average document processing time by day (requires processed_at column from migration 000032)
	timeRows, err := r.pool.Query(ctx, `
		SELECT
			TO_CHAR(DATE(d.uploaded_at), 'YYYY-MM-DD'),
			COALESCE(AVG(EXTRACT(EPOCH FROM (d.processed_at - d.uploaded_at)) / 60.0), 0)
		FROM applicants_document d
		JOIN applicants a ON d.applicant_id = a.id
		WHERE d.status = 'completed'
		  AND d.processed_at IS NOT NULL
		  AND d.uploaded_at >= NOW() - INTERVAL '30 days'
		  AND ($1 = 0 OR a.program_id = $1)
		GROUP BY DATE(d.uploaded_at)
		ORDER BY DATE(d.uploaded_at)
	`, programID)
	if err != nil {
		return s, fmt.Errorf("GetGlobalStats doc_processing: %w", err)
	}
	defer timeRows.Close()

	s.DocProcessingByDay = []usecase.DayProcessing{}
	for timeRows.Next() {
		var dp usecase.DayProcessing
		if err := timeRows.Scan(&dp.Date, &dp.AvgMinutes); err != nil {
			return s, err
		}
		s.DocProcessingByDay = append(s.DocProcessingByDay, dp)
	}

	return s, nil
}

func (r *StatsRepo) GetDynamics(ctx context.Context, period string, programID int64) ([]usecase.DailyStats, error) {
	var datePart, intervalStr, format string

	switch period {
	case "weekly":
		datePart = "DATE_TRUNC('week', created_at)"
		intervalStr = "12 weeks"
		format = "YYYY-MM-DD"
	case "monthly":
		datePart = "DATE_TRUNC('month', created_at)"
		intervalStr = "12 months"
		format = "YYYY-MM"
	default: // daily
		datePart = "DATE(created_at)"
		intervalStr = "30 days"
		format = "YYYY-MM-DD"
	}

	query := fmt.Sprintf(`
		SELECT
			TO_CHAR(%s, '%s')                                        AS date,
			COUNT(*) FILTER (WHERE status IN ('uploaded', 'processing')) AS submitted,
			COUNT(*) FILTER (WHERE status = 'verifying')             AS verifying,
			COUNT(*) FILTER (WHERE status = 'assessed')              AS assessing,
			COUNT(*) FILTER (WHERE status = 'completed')             AS evaluated
		FROM applicants
		WHERE ($1 = 0 OR program_id = $1)
		  AND created_at >= NOW() - INTERVAL '%s'
		GROUP BY %s
		ORDER BY %s
	`, datePart, format, intervalStr, datePart, datePart)

	rows, err := r.pool.Query(ctx, query, programID)
	if err != nil {
		return nil, fmt.Errorf("GetDynamics %s: %w", period, err)
	}
	defer rows.Close()

	result := []usecase.DailyStats{}
	for rows.Next() {
		var ds usecase.DailyStats
		if err := rows.Scan(&ds.Date, &ds.Submitted, &ds.Verifying, &ds.Assessing, &ds.Evaluated); err != nil {
			return nil, err
		}
		result = append(result, ds)
	}
	return result, nil
}
