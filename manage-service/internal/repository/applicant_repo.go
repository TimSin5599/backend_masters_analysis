package repository

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"manage-service/internal/domain/entity"
)

type ApplicantRepo struct {
	pool *pgxpool.Pool
}

func NewApplicantRepo(pool *pgxpool.Pool) *ApplicantRepo {
	return &ApplicantRepo{pool: pool}
}

func (r *ApplicantRepo) Store(ctx context.Context, a *entity.Applicant) error {
	query := `INSERT INTO applicants (program_id, status) VALUES ($1, $2) RETURNING id`
	err := r.pool.QueryRow(ctx, query, a.ProgramID, a.Status).Scan(&a.ID)
	return err
}

func (r *ApplicantRepo) Update(ctx context.Context, a entity.Applicant) error {
	query := `UPDATE applicants SET status=$1, updated_at=NOW() WHERE id=$2`
	_, err := r.pool.Exec(ctx, query, a.Status, a.ID)
	return err
}

func (r *ApplicantRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM applicants WHERE id=$1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *ApplicantRepo) GetByID(ctx context.Context, id int64) (entity.Applicant, error) {
	query := `SELECT id, program_id, status, aggregated_score, created_at, updated_at FROM applicants WHERE id=$1`
	var a entity.Applicant
	err := r.pool.QueryRow(ctx, query, id).Scan(&a.ID, &a.ProgramID, &a.Status, &a.AggregatedScore, &a.CreatedAt, &a.UpdatedAt)
	a.Score = float64(a.AggregatedScore)
	return a, err
}

func (r *ApplicantRepo) StoreDataVersion(ctx context.Context, dv entity.DataVersion) error {
	query := `INSERT INTO data_versions (applicant_id, category, data_content, version_number, source, author_id) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.pool.Exec(ctx, query, dv.ApplicantID, dv.Category, dv.DataContent, dv.VersionNumber, dv.Source, dv.AuthorID)
	return err
}

func (r *ApplicantRepo) StoreIdentification(ctx context.Context, data entity.IdentificationData) error {
	query := `INSERT INTO applicants_data_identification
		(applicant_id, document_id, name, surname, patronymic, email, phone, document_number, date_of_birth, gender, nationality, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := r.pool.Exec(ctx, query,
		data.ApplicantID, data.DocumentID, data.Name, data.Surname, data.Patronymic,
		data.Email, data.Phone, data.DocumentNumber, data.DateOfBirth,
		data.Gender, data.Nationality, data.Source,
	)
	return err
}

func (r *ApplicantRepo) List(ctx context.Context, programID int64) ([]entity.Applicant, error) {
	query := `
		SELECT
			a.id, a.program_id, a.status, a.aggregated_score, a.created_at, a.updated_at,
			COALESCE(i.name, ''), COALESCE(i.surname, ''), COALESCE(i.patronymic, '')
		FROM applicants a
		LEFT JOIN LATERAL (
			SELECT name, surname, patronymic
			FROM applicants_data_identification
			WHERE applicant_id = a.id
			ORDER BY id DESC
			LIMIT 1
		) i ON true
	`
	if programID > 0 {
		query += fmt.Sprintf(" WHERE a.program_id = %d", programID)
	}

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applicants := []entity.Applicant{}
	for rows.Next() {
		var a entity.Applicant
		err = rows.Scan(
			&a.ID, &a.ProgramID, &a.Status, &a.AggregatedScore, &a.CreatedAt, &a.UpdatedAt,
			&a.FirstName, &a.LastName, &a.Patronymic,
		)
		if err != nil {
			return nil, err
		}
		a.Score = float64(a.AggregatedScore)
		applicants = append(applicants, a)
	}
	return applicants, nil
}

func (r *ApplicantRepo) GetDataVersions(ctx context.Context, applicantID int64, category string) ([]entity.DataVersion, error) {
	return nil, nil
}

func (r *ApplicantRepo) StoreEducation(ctx context.Context, data entity.EducationData) error {
	query := `INSERT INTO applicants_data_education (applicant_id, document_id, institution_name, degree_title, major, graduation_date, diploma_serial_number, source) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, query, data.ApplicantID, data.DocumentID, data.InstitutionName, data.DegreeTitle, data.Major, data.GraduationDate, data.DiplomaSerialNumber, data.Source)
	return err
}

func (r *ApplicantRepo) StoreWorkExperience(ctx context.Context, data entity.WorkExperience) error {
	query := `INSERT INTO applicants_data_work_experience (applicant_id, document_id, company_name, position, start_date, end_date, country, city, record_type, competencies, source) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := r.pool.Exec(ctx, query, data.ApplicantID, data.DocumentID, data.CompanyName, data.Position, data.StartDate, data.EndDate, data.Country, data.City, data.RecordType, data.Competencies, data.Source)
	return err
}

func (r *ApplicantRepo) GetLatestIdentification(ctx context.Context, applicantID int64) (entity.IdentificationData, error) {
	query := `
		SELECT
			id, applicant_id, document_id,
			COALESCE(name, ''), COALESCE(surname, ''), COALESCE(patronymic, ''),
			COALESCE(email, ''), COALESCE(phone, ''), COALESCE(document_number, ''),
			COALESCE(date_of_birth, '1970-01-01'),
			COALESCE(gender, ''), COALESCE(nationality, ''),
			source
		FROM applicants_data_identification
		WHERE applicant_id=$1
		ORDER BY id DESC
		LIMIT 1`
	var d entity.IdentificationData
	err := r.pool.QueryRow(ctx, query, applicantID).Scan(
		&d.ID, &d.ApplicantID, &d.DocumentID, &d.Name, &d.Surname, &d.Patronymic, &d.Email, &d.Phone, &d.DocumentNumber, &d.DateOfBirth, &d.Gender, &d.Nationality, &d.Source,
	)
	if err != nil {
		if err.Error() == "no rows in result set" || err.Error() == "pgx: no rows in result set" {
			return d, nil
		}
		return d, err
	}
	return d, nil
}

func (r *ApplicantRepo) GetLatestEducation(ctx context.Context, applicantID int64) (entity.EducationData, error) {
	query := `SELECT id, applicant_id, document_id, institution_name, degree_title, major, graduation_date, diploma_serial_number, source
              FROM applicants_data_education
              WHERE applicant_id=$1 AND document_id IN (SELECT id FROM applicants_document WHERE file_type = 'diploma')
              ORDER BY id DESC LIMIT 1`
	var d entity.EducationData
	err := r.pool.QueryRow(ctx, query, applicantID).Scan(
		&d.ID, &d.ApplicantID, &d.DocumentID, &d.InstitutionName, &d.DegreeTitle, &d.Major, &d.GraduationDate, &d.DiplomaSerialNumber, &d.Source,
	)
	if err != nil {
		if err.Error() == "no rows in result set" || err.Error() == "pgx: no rows in result set" {
			return d, nil
		}
		return d, err
	}
	return d, nil
}

func (r *ApplicantRepo) ListEducation(ctx context.Context, applicantID int64) ([]entity.EducationData, error) {
	query := `SELECT id, applicant_id, document_id, institution_name, degree_title, major, graduation_date, diploma_serial_number, source
              FROM applicants_data_education
              WHERE applicant_id=$1 AND document_id IN (SELECT id FROM applicants_document WHERE file_type = 'second_diploma')
              ORDER BY graduation_date DESC`
	rows, err := r.pool.Query(ctx, query, applicantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]entity.EducationData, 0)
	for rows.Next() {
		var d entity.EducationData
		err = rows.Scan(&d.ID, &d.ApplicantID, &d.DocumentID, &d.InstitutionName, &d.DegreeTitle, &d.Major, &d.GraduationDate, &d.DiplomaSerialNumber, &d.Source)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, nil
}

func (r *ApplicantRepo) ListWorkExperience(ctx context.Context, applicantID int64, fileType string) ([]entity.WorkExperience, error) {
	query := `SELECT id, applicant_id, document_id, COALESCE(country, ''), COALESCE(city, ''), COALESCE(company_name, ''), COALESCE(position, ''), COALESCE(start_date, '1970-01-01'), end_date, COALESCE(record_type, ''), COALESCE(competencies, ''), COALESCE(source, '') FROM applicants_data_work_experience WHERE applicant_id=$1`

	args := []interface{}{applicantID}
	if fileType != "" {
		if fileType == "prof_development" {
			query += ` AND (document_id IN (SELECT id FROM applicants_document WHERE file_type IN ('prof_development', 'internship', 'training', 'resume')) OR document_id IS NULL)`
		} else {
			query += ` AND (document_id IN (SELECT id FROM applicants_document WHERE file_type = $2) OR document_id IS NULL)`
			args = append(args, fileType)
		}
	}

	query += ` ORDER BY start_date DESC`
	log.Printf("[REPO] ListWorkExperience: applicant_id=%d, fileType=%s", applicantID, fileType)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]entity.WorkExperience, 0)
	for rows.Next() {
		var d entity.WorkExperience
		err = rows.Scan(&d.ID, &d.ApplicantID, &d.DocumentID, &d.Country, &d.City, &d.CompanyName, &d.Position, &d.StartDate, &d.EndDate, &d.RecordType, &d.Competencies, &d.Source)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	log.Printf("[REPO] ListWorkExperience: found %d records", len(list))
	return list, nil
}

func (r *ApplicantRepo) ListRecommendations(ctx context.Context, applicantID int64) ([]entity.RecommendationData, error) {
	query := `SELECT id, applicant_id, document_id, author_name, author_position, author_institution, key_strengths, source FROM applicants_data_recommendation WHERE applicant_id=$1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, applicantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []entity.RecommendationData
	for rows.Next() {
		var d entity.RecommendationData
		err = rows.Scan(&d.ID, &d.ApplicantID, &d.DocumentID, &d.AuthorName, &d.AuthorPosition, &d.AuthorInstitution, &d.KeyStrengths, &d.Source)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, nil
}

func (r *ApplicantRepo) ListAchievements(ctx context.Context, applicantID int64, fileType string) ([]entity.AchievementData, error) {
	query := `SELECT id, applicant_id, document_id, COALESCE(achievement_title, ''), COALESCE(description, ''), COALESCE(date_received, '1970-01-01'), COALESCE(document_path, ''), COALESCE(achievement_type, ''), COALESCE(source, '') FROM applicants_data_achievements WHERE applicant_id=$1`

	args := []interface{}{applicantID}
	if fileType != "" {
		if fileType == "achievement" {
			// Exclude certification — it has its own dedicated section
			query += ` AND (document_id IN (SELECT id FROM applicants_document WHERE file_type IN ('achievement', 'resume')) OR document_id IS NULL)`
		} else {
			query += ` AND (document_id IN (SELECT id FROM applicants_document WHERE file_type = $2) OR document_id IS NULL)`
			args = append(args, fileType)
		}
	}

	query += ` ORDER BY date_received DESC`
	log.Printf("[REPO] ListAchievements: applicant_id=%d, fileType=%s", applicantID, fileType)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []entity.AchievementData
	for rows.Next() {
		var d entity.AchievementData
		err = rows.Scan(&d.ID, &d.ApplicantID, &d.DocumentID, &d.AchievementTitle, &d.Description, &d.DateReceived, &d.DocumentPath, &d.AchievementType, &d.Source)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	log.Printf("[REPO] ListAchievements: found %d records", len(list))
	return list, nil
}

func (r *ApplicantRepo) GetLatestTranscript(ctx context.Context, applicantID int64) (entity.TranscriptData, error) {
	query := `SELECT id, applicant_id, document_id,
		COALESCE(cumulative_gpa, 0), COALESCE(cumulative_grade, ''),
		COALESCE(total_credits, 0), COALESCE(obtained_credits, 0),
		COALESCE(total_semesters, 0), source FROM applicants_data_transcript WHERE applicant_id=$1 ORDER BY id DESC LIMIT 1`
	var d entity.TranscriptData
	err := r.pool.QueryRow(ctx, query, applicantID).Scan(
		&d.ID, &d.ApplicantID, &d.DocumentID, &d.CumulativeGPA, &d.CumulativeGrade,
		&d.TotalCredits, &d.ObtainedCredits, &d.TotalSemesters, &d.Source,
	)
	if err != nil {
		if err.Error() == "no rows in result set" || err.Error() == "pgx: no rows in result set" {
			return d, nil
		}
		return d, err
	}
	return d, nil
}

func (r *ApplicantRepo) GetLatestLanguageTraining(ctx context.Context, applicantID int64) (entity.LanguageTraining, error) {
	query := `SELECT id, applicant_id, document_id, russian_level, english_level, certificate_path, COALESCE(exam_name, ''), COALESCE(score, ''), source FROM applicants_data_language_training WHERE applicant_id=$1 ORDER BY id DESC LIMIT 1`
	var d entity.LanguageTraining
	err := r.pool.QueryRow(ctx, query, applicantID).Scan(&d.ID, &d.ApplicantID, &d.DocumentID, &d.RussianLevel, &d.EnglishLevel, &d.CertificatePath, &d.ExamName, &d.Score, &d.Source)
	if err != nil {
		if err.Error() == "no rows in result set" || err.Error() == "pgx: no rows in result set" {
			return d, nil
		}
		return d, err
	}
	return d, nil
}

func (r *ApplicantRepo) StoreRecommendation(ctx context.Context, data entity.RecommendationData) error {
	query := `INSERT INTO applicants_data_recommendation (applicant_id, document_id, author_name, author_position, author_institution, key_strengths, source) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.pool.Exec(ctx, query, data.ApplicantID, data.DocumentID, data.AuthorName, data.AuthorPosition, data.AuthorInstitution, data.KeyStrengths, data.Source)
	return err
}

func (r *ApplicantRepo) StoreAchievement(ctx context.Context, data entity.AchievementData) error {
	query := `INSERT INTO applicants_data_achievements (applicant_id, document_id, achievement_title, description, date_received, document_path, achievement_type, source) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, query, data.ApplicantID, data.DocumentID, data.AchievementTitle, data.Description, data.DateReceived, data.DocumentPath, data.AchievementType, data.Source)
	return err
}

func (r *ApplicantRepo) StoreLanguageTraining(ctx context.Context, data entity.LanguageTraining) error {
	query := `INSERT INTO applicants_data_language_training (applicant_id, document_id, russian_level, english_level, certificate_path, exam_name, score, source) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, query, data.ApplicantID, data.DocumentID, data.RussianLevel, data.EnglishLevel, data.CertificatePath, data.ExamName, data.Score, data.Source)
	return err
}

func (r *ApplicantRepo) StoreMotivation(ctx context.Context, data entity.MotivationData) error {
	query := `INSERT INTO applicants_data_motivation (applicant_id, document_id, reasons_for_applying, experience_summary, career_goals, detected_language, main_text, source) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, query, data.ApplicantID, data.DocumentID, data.ReasonsForApplying, data.ExperienceSummary, data.CareerGoals, data.DetectedLanguage, data.MainText, data.Source)
	return err
}

func (r *ApplicantRepo) GetLatestMotivation(ctx context.Context, applicantID int64) (entity.MotivationData, error) {
	query := `SELECT id, applicant_id, document_id, COALESCE(reasons_for_applying, ''), COALESCE(experience_summary, ''), COALESCE(career_goals, ''), COALESCE(detected_language, ''), COALESCE(main_text, ''), source FROM applicants_data_motivation WHERE applicant_id=$1 ORDER BY id DESC LIMIT 1`
	var d entity.MotivationData
	err := r.pool.QueryRow(ctx, query, applicantID).Scan(&d.ID, &d.ApplicantID, &d.DocumentID, &d.ReasonsForApplying, &d.ExperienceSummary, &d.CareerGoals, &d.DetectedLanguage, &d.MainText, &d.Source)
	if err != nil {
		if err.Error() == "no rows in result set" || err.Error() == "pgx: no rows in result set" {
			return d, nil
		}
		return d, err
	}
	return d, nil
}

func (r *ApplicantRepo) StoreTranscript(ctx context.Context, data entity.TranscriptData) error {
	query := `INSERT INTO applicants_data_transcript
		(applicant_id, document_id, cumulative_gpa, cumulative_grade, total_credits, obtained_credits, total_semesters, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (document_id) DO UPDATE SET
			applicant_id = EXCLUDED.applicant_id,
			cumulative_gpa = EXCLUDED.cumulative_gpa,
			cumulative_grade = EXCLUDED.cumulative_grade,
			total_credits = EXCLUDED.total_credits,
			obtained_credits = EXCLUDED.obtained_credits,
			total_semesters = EXCLUDED.total_semesters,
			source = EXCLUDED.source,
			created_at = NOW()`
	_, err := r.pool.Exec(ctx, query,
		data.ApplicantID, data.DocumentID, data.CumulativeGPA, data.CumulativeGrade,
		data.TotalCredits, data.ObtainedCredits, data.TotalSemesters, data.Source)
	return err
}

func (r *ApplicantRepo) GetTranscriptByDocumentID(ctx context.Context, documentID int64) (entity.TranscriptData, error) {
	query := `SELECT id, applicant_id, document_id,
		cumulative_gpa, cumulative_grade, total_credits, obtained_credits,
		total_semesters, source FROM applicants_data_transcript WHERE document_id=$1 LIMIT 1`
	var d entity.TranscriptData
	err := r.pool.QueryRow(ctx, query, documentID).Scan(
		&d.ID, &d.ApplicantID, &d.DocumentID, &d.CumulativeGPA, &d.CumulativeGrade,
		&d.TotalCredits, &d.ObtainedCredits, &d.TotalSemesters, &d.Source,
	)
	return d, err
}

func (r *ApplicantRepo) DeleteWorkExperience(ctx context.Context, id int64) error {
	query := `DELETE FROM applicants_data_work_experience WHERE id=$1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *ApplicantRepo) DeleteRecommendation(ctx context.Context, id int64) error {
	query := `DELETE FROM applicants_data_recommendation WHERE id=$1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *ApplicantRepo) DeleteAchievement(ctx context.Context, id int64) error {
	query := `DELETE FROM applicants_data_achievements WHERE id=$1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *ApplicantRepo) DeleteLanguageTraining(ctx context.Context, id int64) error {
	query := `DELETE FROM applicants_data_language_training WHERE id=$1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *ApplicantRepo) UpdateIdentification(ctx context.Context, data entity.IdentificationData) error {
	query := `UPDATE applicants_data_identification SET
		name=$1, surname=$2, patronymic=$3, email=$4, phone=$5,
		document_number=$6, date_of_birth=$7, gender=$8, nationality=$9,
		source=$10
		WHERE id=$11`
	_, err := r.pool.Exec(ctx, query,
		data.Name, data.Surname, data.Patronymic, data.Email, data.Phone,
		data.DocumentNumber, data.DateOfBirth, data.Gender, data.Nationality,
		data.Source, data.ID,
	)
	return err
}

func (r *ApplicantRepo) UpdateEducation(ctx context.Context, data entity.EducationData) error {
	query := `UPDATE applicants_data_education SET
		institution_name=$1, degree_title=$2, major=$3, graduation_date=$4,
		diploma_serial_number=$5, source=$6
		WHERE id=$7`
	_, err := r.pool.Exec(ctx, query,
		data.InstitutionName, data.DegreeTitle, data.Major, data.GraduationDate,
		data.DiplomaSerialNumber, data.Source, data.ID,
	)
	return err
}

func (r *ApplicantRepo) UpdateTranscript(ctx context.Context, data entity.TranscriptData) error {
	query := `UPDATE applicants_data_transcript SET
		cumulative_gpa=$1, cumulative_grade=$2, total_credits=$3,
		obtained_credits=$4, total_semesters=$5, source=$6
		WHERE id=$7`
	_, err := r.pool.Exec(ctx, query,
		data.CumulativeGPA, data.CumulativeGrade, data.TotalCredits,
		data.ObtainedCredits, data.TotalSemesters, data.Source, data.ID,
	)
	return err
}

func (r *ApplicantRepo) UpdateWorkExperience(ctx context.Context, data entity.WorkExperience) error {
	query := `UPDATE applicants_data_work_experience SET
		company_name=$1, position=$2, start_date=$3, end_date=$4,
		country=$5, city=$6, record_type=$7, competencies=$8, source=$9
		WHERE id=$10`
	_, err := r.pool.Exec(ctx, query,
		data.CompanyName, data.Position, data.StartDate, data.EndDate,
		data.Country, data.City, data.RecordType, data.Competencies, data.Source, data.ID,
	)
	return err
}

func (r *ApplicantRepo) UpdateRecommendation(ctx context.Context, data entity.RecommendationData) error {
	query := `UPDATE applicants_data_recommendation SET
		author_name=$1, author_position=$2, author_institution=$3,
		key_strengths=$4, source=$5
		WHERE id=$6`
	_, err := r.pool.Exec(ctx, query,
		data.AuthorName, data.AuthorPosition, data.AuthorInstitution,
		data.KeyStrengths, data.Source, data.ID,
	)
	return err
}

func (r *ApplicantRepo) UpdateAchievement(ctx context.Context, data entity.AchievementData) error {
	query := `UPDATE applicants_data_achievements SET
		achievement_title=$1, description=$2, date_received=$3,
		achievement_type=$4, source=$5
		WHERE id=$6`
	_, err := r.pool.Exec(ctx, query,
		data.AchievementTitle, data.Description, data.DateReceived,
		data.AchievementType, data.Source, data.ID,
	)
	return err
}

func (r *ApplicantRepo) UpdateLanguageTraining(ctx context.Context, data entity.LanguageTraining) error {
	query := `UPDATE applicants_data_language_training SET
		russian_level=$1, english_level=$2, exam_name=$3,
		score=$4, source=$5
		WHERE id=$6`
	_, err := r.pool.Exec(ctx, query,
		data.RussianLevel, data.EnglishLevel, data.ExamName,
		data.Score, data.Source, data.ID,
	)
	return err
}

func (r *ApplicantRepo) UpdateMotivation(ctx context.Context, data entity.MotivationData) error {
	query := `UPDATE applicants_data_motivation SET
		reasons_for_applying=$1, experience_summary=$2, career_goals=$3,
		detected_language=$4, main_text=$5, source=$6
		WHERE id=$7`
	_, err := r.pool.Exec(ctx, query,
		data.ReasonsForApplying, data.ExperienceSummary, data.CareerGoals,
		data.DetectedLanguage, data.MainText, data.Source, data.ID,
	)
	return err
}

func (r *ApplicantRepo) DeleteDataByDocumentID(ctx context.Context, category string, documentID int64) error {
	var query string
	switch category {
	case "personal_data":
		query = `DELETE FROM applicants_data_identification WHERE document_id=$1`
	case "diploma", "education":
		query = `DELETE FROM applicants_data_education WHERE document_id=$1`
	case "recommendation":
		query = `DELETE FROM applicants_data_recommendation WHERE document_id=$1`
	case "achievement":
		query = `DELETE FROM applicants_data_achievements WHERE document_id=$1`
	case "language":
		query = `DELETE FROM applicants_data_language_training WHERE document_id=$1`
	case "motivation":
		query = `DELETE FROM applicants_data_motivation WHERE document_id=$1`
	case "work", "prof_development", "internship", "training":
		query = `DELETE FROM applicants_data_work_experience WHERE document_id=$1`
	case "transcript":
		query = `DELETE FROM applicants_data_transcript WHERE document_id=$1`
	case "second_diploma":
		query = `DELETE FROM applicants_data_education WHERE document_id=$1`
	case "certification":
		query = `DELETE FROM applicants_data_achievements WHERE document_id=$1`
	case "resume":
		_, _ = r.pool.Exec(ctx, `DELETE FROM applicants_data_identification WHERE document_id=$1`, documentID)
		_, _ = r.pool.Exec(ctx, `DELETE FROM applicants_data_work_experience WHERE document_id=$1`, documentID)
		_, _ = r.pool.Exec(ctx, `DELETE FROM applicants_data_achievements WHERE document_id=$1`, documentID)
		return nil
	default:
		return nil
	}
	_, err := r.pool.Exec(ctx, query, documentID)
	return err
}

func (r *ApplicantRepo) UpdateApplicantRanking(ctx context.Context, applicantID int64, score float64, status string) error {
	query := `UPDATE applicants SET aggregated_score=$1, status=$2, updated_at=NOW() WHERE id=$3`
	_, err := r.pool.Exec(ctx, query, score, status, applicantID)
	return err
}

func (r *ApplicantRepo) GetLatestVideo(ctx context.Context, applicantID int64) (entity.VideoData, error) {
	query := `SELECT id, applicant_id, video_url, source, created_at FROM applicants_data_video WHERE applicant_id=$1 ORDER BY id DESC LIMIT 1`
	var d entity.VideoData
	err := r.pool.QueryRow(ctx, query, applicantID).Scan(&d.ID, &d.ApplicantID, &d.VideoURL, &d.Source, &d.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return d, nil
		}
		return d, err
	}
	return d, nil
}

func (r *ApplicantRepo) UpdateVideo(ctx context.Context, data entity.VideoData) error {
	query := `INSERT INTO applicants_data_video (applicant_id, video_url, source) VALUES ($1, $2, $3)`
	_, err := r.pool.Exec(ctx, query, data.ApplicantID, data.VideoURL, data.Source)
	return err
}

func (r *ApplicantRepo) GetScoringScheme(ctx context.Context, applicantID int64) (string, error) {
	var scheme string
	err := r.pool.QueryRow(ctx, `SELECT scoring_scheme FROM applicants WHERE id=$1`, applicantID).Scan(&scheme)
	if err != nil {
		return "auto", err
	}
	return scheme, nil
}

func (r *ApplicantRepo) SetScoringScheme(ctx context.Context, applicantID int64, scheme string) error {
	_, err := r.pool.Exec(ctx, `UPDATE applicants SET scoring_scheme=$1 WHERE id=$2`, scheme, applicantID)
	return err
}

// ConfirmModelData updates source from 'model' to confirmedBy across all data tables for the applicant.
func (r *ApplicantRepo) ConfirmModelData(ctx context.Context, applicantID int64, confirmedBy string) error {
	tables := []string{
		"applicants_data_identification",
		"applicants_data_education",
		"applicants_data_transcript",
		"applicants_data_work_experience",
		"applicants_data_language_training",
		"applicants_data_motivation",
		"applicants_data_recommendation",
		"applicants_data_achievements",
		"applicants_data_resume",
		"applicants_data_video",
	}
	for _, table := range tables {
		q := fmt.Sprintf(`UPDATE %s SET source=$1 WHERE applicant_id=$2 AND source='model'`, table)
		if _, err := r.pool.Exec(ctx, q, confirmedBy, applicantID); err != nil {
			return fmt.Errorf("ConfirmModelData(%s): %w", table, err)
		}
	}
	return nil
}
