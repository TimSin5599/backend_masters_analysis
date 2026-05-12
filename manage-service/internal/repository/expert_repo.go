package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"manage-service/internal/domain/entity"
)

type ExpertRepo struct {
	pool *pgxpool.Pool
}

func NewExpertRepo(pool *pgxpool.Pool) *ExpertRepo {
	return &ExpertRepo{pool: pool}
}

func (r *ExpertRepo) StoreEvaluation(ctx context.Context, eval entity.ExpertEvaluation) error {
	query := `INSERT INTO expert_evaluations
		(applicant_id, expert_id, category, score, comment, status, updated_by_id, is_admin_override, is_ai_generated, source_info)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.pool.Exec(ctx, query,
		eval.ApplicantID, eval.ExpertID, eval.Category, eval.Score, eval.Comment, eval.Status,
		eval.UpdatedByID, eval.IsAdminOverride, eval.IsAIGenerated, eval.SourceInfo,
	)
	return err
}

func (r *ExpertRepo) UpdateEvaluation(ctx context.Context, eval entity.ExpertEvaluation) error {
	query := `UPDATE expert_evaluations SET
		score=$1, comment=$2, status=$3, updated_by_id=$4, is_admin_override=$5, is_ai_generated=$6, source_info=$7, updated_at=NOW()
		WHERE applicant_id=$8 AND expert_id=$9 AND category=$10`
	_, err := r.pool.Exec(ctx, query,
		eval.Score, eval.Comment, eval.Status, eval.UpdatedByID, eval.IsAdminOverride, eval.IsAIGenerated, eval.SourceInfo,
		eval.ApplicantID, eval.ExpertID, eval.Category,
	)
	return err
}

func (r *ExpertRepo) UpdateEvaluationStatus(ctx context.Context, applicantID int64, expertID string, status string) error {
	query := `UPDATE expert_evaluations SET status=$1, updated_at=NOW() WHERE applicant_id=$2 AND expert_id=$3`
	_, err := r.pool.Exec(ctx, query, status, applicantID, expertID)
	return err
}

func (r *ExpertRepo) ListEvaluations(ctx context.Context, applicantID int64) ([]entity.ExpertEvaluation, error) {
	query := `SELECT id, applicant_id, expert_id, category, score, comment, status, updated_by_id, is_admin_override, is_ai_generated, source_info, created_at, updated_at
		FROM expert_evaluations WHERE applicant_id=$1`
	rows, err := r.pool.Query(ctx, query, applicantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	evals := make([]entity.ExpertEvaluation, 0)
	for rows.Next() {
		var e entity.ExpertEvaluation
		err := rows.Scan(&e.ID, &e.ApplicantID, &e.ExpertID, &e.Category, &e.Score, &e.Comment, &e.Status, &e.UpdatedByID, &e.IsAdminOverride, &e.IsAIGenerated, &e.SourceInfo, &e.CreatedAt, &e.UpdatedAt)
		if err != nil {
			return nil, err
		}
		evals = append(evals, e)
	}
	return evals, nil
}

func (r *ExpertRepo) GetEvaluation(ctx context.Context, applicantID int64, expertID string, category string) (entity.ExpertEvaluation, error) {
	query := `SELECT id, applicant_id, expert_id, category, score, comment, status, updated_by_id, is_admin_override, is_ai_generated, source_info, created_at, updated_at
		FROM expert_evaluations WHERE applicant_id = $1 AND expert_id = $2 AND category = $3`
	var e entity.ExpertEvaluation
	err := r.pool.QueryRow(ctx, query, applicantID, expertID, category).Scan(
		&e.ID, &e.ApplicantID, &e.ExpertID, &e.Category, &e.Score, &e.Comment, &e.Status, &e.UpdatedByID, &e.IsAdminOverride, &e.IsAIGenerated, &e.SourceInfo, &e.CreatedAt, &e.UpdatedAt,
	)
	return e, err
}

func (r *ExpertRepo) GetCriteria(ctx context.Context) ([]entity.EvaluationCriteria, error) {
	query := `SELECT code, title, max_score, type, document_types, is_mandatory, scheme, program_id
	          FROM evaluation_criteria ORDER BY scheme, type, code`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var criteria []entity.EvaluationCriteria
	for rows.Next() {
		var c entity.EvaluationCriteria
		if err := rows.Scan(&c.Code, &c.Title, &c.MaxScore, &c.Type, &c.DocumentTypes, &c.IsMandatory, &c.Scheme, &c.ProgramID); err != nil {
			return nil, err
		}
		criteria = append(criteria, c)
	}
	return criteria, nil
}

func (r *ExpertRepo) CreateCriteria(ctx context.Context, c entity.EvaluationCriteria) error {
	query := `INSERT INTO evaluation_criteria (code, title, max_score, type, document_types, is_mandatory, scheme, program_id)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, query, c.Code, c.Title, c.MaxScore, c.Type, c.DocumentTypes, c.IsMandatory, c.Scheme, c.ProgramID)
	return err
}

func (r *ExpertRepo) UpdateCriteria(ctx context.Context, c entity.EvaluationCriteria) error {
	query := `UPDATE evaluation_criteria
	          SET title=$1, max_score=$2, type=$3, document_types=$4, is_mandatory=$5, scheme=$6, program_id=$7
	          WHERE code=$8`
	_, err := r.pool.Exec(ctx, query, c.Title, c.MaxScore, c.Type, c.DocumentTypes, c.IsMandatory, c.Scheme, c.ProgramID, c.Code)
	return err
}

func (r *ExpertRepo) DeleteCriteria(ctx context.Context, code string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM evaluation_criteria WHERE code=$1`, code)
	return err
}

func (r *ExpertRepo) SaveEvaluationBatch(ctx context.Context, evaluations []entity.ExpertEvaluation) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, eval := range evaluations {
		query := `INSERT INTO expert_evaluations
			(applicant_id, expert_id, category, score, comment, status, updated_by_id, is_admin_override, is_ai_generated, source_info)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (applicant_id, expert_id, category) DO UPDATE SET
			score=EXCLUDED.score, comment=EXCLUDED.comment, status=EXCLUDED.status,
			updated_by_id=EXCLUDED.updated_by_id, is_admin_override=EXCLUDED.is_admin_override,
			is_ai_generated=EXCLUDED.is_ai_generated, source_info=EXCLUDED.source_info, updated_at=NOW()`

		_, err := tx.Exec(ctx, query,
			eval.ApplicantID, eval.ExpertID, eval.Category, eval.Score, eval.Comment, eval.Status,
			eval.UpdatedByID, eval.IsAdminOverride, eval.IsAIGenerated, eval.SourceInfo,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *ExpertRepo) GetAggregatedScore(ctx context.Context, applicantID int64, categories []string) (float64, error) {
	// Rule: If ENGLISH score is 0, total score is 0 for that expert.
	// We need to calculate average of (sum of BASE scores + ALTERNATIVE scores) per expert,
	// but applying the Zero English rule per expert.
	// We only sum categories that are valid for the current applicant scheme.

	query := `
		WITH expert_sums AS (
			SELECT
				expert_id,
				SUM(score) as total_score,
				MAX(CASE WHEN category = 'ENGLISH' AND score = 0 THEN 1 ELSE 0 END) as is_english_zero
			FROM expert_evaluations
			WHERE applicant_id = $1 AND status = 'COMPLETED' AND category = ANY($2)
			  AND expert_id != 'AI_SYSTEM'
			GROUP BY expert_id
		)
		SELECT COALESCE(AVG(CASE WHEN is_english_zero = 1 THEN 0 ELSE total_score END), 0)
		FROM expert_sums
	`
	var avgScore float64
	err := r.pool.QueryRow(ctx, query, applicantID, categories).Scan(&avgScore)
	return avgScore, err
}

func (r *ExpertRepo) GetExpertSlots(ctx context.Context, programID int64) ([]entity.ExpertSlot, error) {
	if programID <= 0 {
		return []entity.ExpertSlot{}, nil
	}
	query := `
		SELECT s.user_id, s.slot_number, s.program_id, u.first_name, u.last_name, s.created_at
		FROM expert_slots s
		LEFT JOIN users u ON s.user_id = u.id
		WHERE s.program_id = $1
		ORDER BY s.slot_number ASC`
	rows, err := r.pool.Query(ctx, query, programID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slots := make([]entity.ExpertSlot, 0)
	for rows.Next() {
		var s entity.ExpertSlot
		// Note: first_name and last_name might be NULL if user is not found (LEFT JOIN)
		var firstName, lastName *string
		err := rows.Scan(&s.UserID, &s.SlotNumber, &s.ProgramID, &firstName, &lastName, &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		if firstName != nil {
			s.FirstName = *firstName
		}
		if lastName != nil {
			s.LastName = *lastName
		}
		slots = append(slots, s)
	}
	return slots, nil
}

func (r *ExpertRepo) AssignExpertSlot(ctx context.Context, userID string, slotNumber int, programID int64) error {
	fmt.Printf("[DEBUG] Repo AssignExpertSlot: userID=%s, slotNumber=%d, programID=%d\n", userID, slotNumber, programID)

	// Delete any existing evaluations for this user (they are being reassigned)
	// and delete evaluations for the user who was in this slot previously (scoped to this program via applicants join)
	_, err := r.pool.Exec(ctx, `
		DELETE FROM expert_evaluations
		WHERE expert_id = $1
		   OR expert_id = (SELECT user_id FROM expert_slots WHERE slot_number = $2 AND program_id = $3)
	`, userID, slotNumber, programID)
	if err != nil {
		fmt.Printf("[ERROR] Failed to clear old evaluations on slot reassignment: %v\n", err)
	}

	query := `INSERT INTO expert_slots (user_id, slot_number, program_id) VALUES ($1, $2, $3)
		ON CONFLICT (slot_number, program_id) DO UPDATE SET user_id = EXCLUDED.user_id`
	_, err = r.pool.Exec(ctx, query, userID, slotNumber, programID)
	return err
}

func (r *ExpertRepo) RemoveExpertSlot(ctx context.Context, slotNumber int, programID int64) error {
	query := `DELETE FROM expert_slots WHERE slot_number=$1 AND program_id=$2`
	_, err := r.pool.Exec(ctx, query, slotNumber, programID)
	return err
}

func (r *ExpertRepo) GetExpertSlotByUserID(ctx context.Context, userID string, programID int64) (entity.ExpertSlot, error) {
	query := `SELECT user_id, slot_number, program_id, created_at FROM expert_slots WHERE user_id=$1 AND program_id=$2`
	var s entity.ExpertSlot
	err := r.pool.QueryRow(ctx, query, userID, programID).Scan(&s.UserID, &s.SlotNumber, &s.ProgramID, &s.CreatedAt)
	return s, err
}

func (r *ExpertRepo) GetUsersByRoles(ctx context.Context, roles []string) ([]entity.User, error) {
	query := `SELECT id, first_name, last_name, email, roles FROM users WHERE roles && $1::text[] ORDER BY first_name ASC`
	rows, err := r.pool.Query(ctx, query, roles)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []entity.User
	for rows.Next() {
		var u entity.User
		err := rows.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Roles)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}
