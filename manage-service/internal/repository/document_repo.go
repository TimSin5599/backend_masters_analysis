package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"manage-service/internal/domain/entity"
)

type DocumentRepo struct {
	pool *pgxpool.Pool
}

func NewDocumentRepo(pool *pgxpool.Pool) *DocumentRepo {
	return &DocumentRepo{pool: pool}
}

func (r *DocumentRepo) StoreDocument(ctx context.Context, d *entity.Document) error {
	query := `INSERT INTO applicants_document (applicant_id, file_type, file_name, storage_path, status) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err := r.pool.QueryRow(ctx, query, d.ApplicantID, d.FileType, d.FileName, d.StoragePath, d.Status).Scan(&d.ID)
	return err
}

func (r *DocumentRepo) UpdateDocumentStatus(ctx context.Context, id int64, status string) error {
	query := `UPDATE applicants_document
		SET status = $1,
		    processed_at = CASE WHEN $2 IN ('completed', 'extraction_failed', 'classification_failed')
		                        THEN NOW()
		                        ELSE processed_at
		                   END
		WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, status, status, id)
	return err
}

func (r *DocumentRepo) MarkProcessingStarted(ctx context.Context, id int64) error {
	query := `UPDATE applicants_document SET processing_started_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *DocumentRepo) UpdateDocumentCategory(ctx context.Context, id int64, category string) error {
	query := `UPDATE applicants_document SET file_type=$1 WHERE id=$2`
	_, err := r.pool.Exec(ctx, query, category, id)
	return err
}

func (r *DocumentRepo) UpdateDocumentStoragePath(ctx context.Context, id int64, storagePath string) error {
	query := `UPDATE applicants_document SET storage_path=$1 WHERE id=$2`
	_, err := r.pool.Exec(ctx, query, storagePath, id)
	return err
}

func (r *DocumentRepo) GetDocuments(ctx context.Context, applicantID int64) ([]entity.Document, error) {
	query := `SELECT id, applicant_id, file_type, file_name, storage_path, status, uploaded_at FROM applicants_document WHERE applicant_id=$1`
	rows, err := r.pool.Query(ctx, query, applicantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []entity.Document
	for rows.Next() {
		var d entity.Document
		err = rows.Scan(&d.ID, &d.ApplicantID, &d.FileType, &d.FileName, &d.StoragePath, &d.Status, &d.UploadedAt)
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, nil
}

func (r *DocumentRepo) GetDocumentByID(ctx context.Context, id int64) (entity.Document, error) {
	query := `SELECT id, applicant_id, file_type, file_name, storage_path, status, uploaded_at FROM applicants_document WHERE id=$1`
	var d entity.Document
	err := r.pool.QueryRow(ctx, query, id).Scan(&d.ID, &d.ApplicantID, &d.FileType, &d.FileName, &d.StoragePath, &d.Status, &d.UploadedAt)
	return d, err
}

func (r *DocumentRepo) StoreExtractedField(ctx context.Context, applicantID int64, documentID int64, field, value string) error {
	query := `INSERT INTO extracted_fields (applicant_id, document_id, field_name, field_value) VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(ctx, query, applicantID, documentID, field, value)
	return err
}

func (r *DocumentRepo) GetLatestDocumentByCategory(ctx context.Context, applicantID int64, category string) (entity.Document, error) {
	// file_type in DB matches category (e.g. 'personal_data', 'diploma')
	query := `SELECT id, applicant_id, file_type, file_name, storage_path, status, uploaded_at FROM applicants_document 
              WHERE applicant_id=$1 AND file_type=$2 
              ORDER BY uploaded_at DESC LIMIT 1`
	var d entity.Document
	err := r.pool.QueryRow(ctx, query, applicantID, category).Scan(&d.ID, &d.ApplicantID, &d.FileType, &d.FileName, &d.StoragePath, &d.Status, &d.UploadedAt)
	return d, err
}

func (r *DocumentRepo) DeleteDocument(ctx context.Context, id int64) error {
	query := `DELETE FROM applicants_document WHERE id=$1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// UpdateDocumentStatusByPath resets the status of the document with the given storage path.
// Used by the recovery worker to un-stick documents whose associated queue task was re-enqueued.
func (r *DocumentRepo) UpdateDocumentStatusByPath(ctx context.Context, storagePath string, status string) error {
	query := `UPDATE applicants_document SET status=$1 WHERE storage_path=$2`
	_, err := r.pool.Exec(ctx, query, status, storagePath)
	return err
}
