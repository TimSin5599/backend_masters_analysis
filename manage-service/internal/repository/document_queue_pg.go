package repository

import (
	"context"
	"fmt"
	"manage-service/internal/domain/entity"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DocumentQueueRepo struct {
	*pgxpool.Pool
}

func NewDocumentQueueRepo(pg *pgxpool.Pool) *DocumentQueueRepo {
	return &DocumentQueueRepo{pg}
}

func (r *DocumentQueueRepo) Enqueue(ctx context.Context, task entity.DocumentQueueTask) (string, error) {
	sql := `INSERT INTO document_processing_queue (applicant_id, document_category, file_path, priority, status)
            VALUES ($1, $2, $3, $4, $5) RETURNING id`
	var id string
	err := r.Pool.QueryRow(ctx, sql, task.ApplicantID, task.DocumentCategory, task.FilePath, task.Priority, task.Status).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("DocumentQueueRepo.Enqueue: %w", err)
	}
	return id, nil
}

func (r *DocumentQueueRepo) UpdateStatus(ctx context.Context, id string, status string, errMsg *string) error {
	sql := `UPDATE document_processing_queue SET status = $1, error_message = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`
	_, err := r.Pool.Exec(ctx, sql, status, errMsg, id)
	if err != nil {
		return fmt.Errorf("DocumentQueueRepo.UpdateStatus: %w", err)
	}
	return nil
}

// GetStuckTasks returns tasks with status "processing" or "pending" that haven't
// been updated for longer than olderThan. Used by the recovery worker on startup.
func (r *DocumentQueueRepo) GetStuckTasks(ctx context.Context, olderThan time.Duration) ([]entity.DocumentQueueTask, error) {
	cutoff := time.Now().Add(-olderThan)
	sql := `SELECT id, applicant_id, document_category, file_path, priority, status, error_message, created_at, updated_at
	        FROM document_processing_queue
	        WHERE status IN ('processing', 'pending')
	        AND updated_at < $1
	        ORDER BY priority DESC, created_at ASC`

	rows, err := r.Pool.Query(ctx, sql, cutoff)
	if err != nil {
		return nil, fmt.Errorf("DocumentQueueRepo.GetStuckTasks: %w", err)
	}
	defer rows.Close()

	var tasks []entity.DocumentQueueTask
	for rows.Next() {
		var t entity.DocumentQueueTask
		if err := rows.Scan(&t.ID, &t.ApplicantID, &t.DocumentCategory, &t.FilePath, &t.Priority, &t.Status, &t.ErrorMessage, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("DocumentQueueRepo.GetStuckTasks row scan: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (r *DocumentQueueRepo) GetByApplicantID(ctx context.Context, applicantID int64) ([]entity.DocumentQueueTask, error) {
	sql := `SELECT id, applicant_id, document_category, file_path, priority, status, error_message, created_at, updated_at 
            FROM document_processing_queue WHERE applicant_id = $1 ORDER BY created_at ASC`
	rows, err := r.Pool.Query(ctx, sql, applicantID)
	if err != nil {
		return nil, fmt.Errorf("DocumentQueueRepo.GetByApplicantID: %w", err)
	}
	defer rows.Close()

	var tasks []entity.DocumentQueueTask
	for rows.Next() {
		var t entity.DocumentQueueTask
		err := rows.Scan(&t.ID, &t.ApplicantID, &t.DocumentCategory, &t.FilePath, &t.Priority, &t.Status, &t.ErrorMessage, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("DocumentQueueRepo.GetByApplicantID row scan: %w", err)
		}
		tasks = append(tasks, t)
	}

	return tasks, nil
}
