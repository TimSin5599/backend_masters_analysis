package repository

import (
	"context"
	"fmt"
	"manage-service/internal/entity"

	"github.com/jackc/pgx/v5/pgxpool"
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
	query := `SELECT id, program_id, status, created_at, updated_at FROM applicants WHERE id=$1`
	var a entity.Applicant
	err := r.pool.QueryRow(ctx, query, id).Scan(&a.ID, &a.ProgramID, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func (r *ApplicantRepo) StoreDocument(ctx context.Context, d *entity.Document) error {
	query := `INSERT INTO applicants_document (applicant_id, file_type, file_name, storage_path) VALUES ($1, $2, $3, $4) RETURNING id`
	err := r.pool.QueryRow(ctx, query, d.ApplicantID, d.FileType, d.FileName, d.StoragePath).Scan(&d.ID)
	return err
}

func (r *ApplicantRepo) UpdateDocumentStatus(ctx context.Context, id int64, status string) error {
	query := `UPDATE applicants_document SET status=$1 WHERE id=$2`
	_, err := r.pool.Exec(ctx, query, status, id)
	return err
}

func (r *ApplicantRepo) StoreDataVersion(ctx context.Context, dv entity.DataVersion) error {
	query := `INSERT INTO data_versions (applicant_id, category, data_content, version_number, source, author_id) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.pool.Exec(ctx, query, dv.ApplicantID, dv.Category, dv.DataContent, dv.VersionNumber, dv.Source, dv.AuthorID)
	return err
}

func (r *ApplicantRepo) StoreIdentification(ctx context.Context, data entity.IdentificationData) error {
	query := `INSERT INTO applicants_data_identification 
		(applicant_id, document_id, name, surname, patronymic, email, phone, document_number, date_of_birth, gender, nationality, source, version) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err := r.pool.Exec(ctx, query, 
		data.ApplicantID, data.DocumentID, data.Name, data.Surname, data.Patronymic, 
		data.Email, data.Phone, data.DocumentNumber, data.DateOfBirth, 
		data.Gender, data.Nationality, data.Source, data.Version,
	)
	return err
}

func (r *ApplicantRepo) List(ctx context.Context, programID int64) ([]entity.Applicant, error) {
	query := `
		SELECT 
			a.id, a.program_id, a.status, a.created_at, a.updated_at,
			COALESCE(i.name, ''), COALESCE(i.surname, ''), COALESCE(i.patronymic, '')
		FROM applicants a
		LEFT JOIN LATERAL (
			SELECT name, surname, patronymic 
			FROM applicants_data_identification 
			WHERE applicant_id = a.id 
			ORDER BY version DESC 
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
			&a.ID, &a.ProgramID, &a.Status, &a.CreatedAt, &a.UpdatedAt,
			&a.FirstName, &a.LastName, &a.Patronymic,
		)
		if err != nil {
			return nil, err
		}
		a.Score = 0.0 // Placeholder until score calculation is implemented
		applicants = append(applicants, a)
	}
	return applicants, nil
}

func (r *ApplicantRepo) GetDocuments(ctx context.Context, applicantID int64) ([]entity.Document, error) {
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

func (r *ApplicantRepo) GetDocumentByID(ctx context.Context, id int64) (entity.Document, error) {
	query := `SELECT id, applicant_id, file_type, file_name, storage_path, status, uploaded_at FROM applicants_document WHERE id=$1`
	var d entity.Document
	err := r.pool.QueryRow(ctx, query, id).Scan(&d.ID, &d.ApplicantID, &d.FileType, &d.FileName, &d.StoragePath, &d.Status, &d.UploadedAt)
	return d, err
}

func (r *ApplicantRepo) GetDataVersions(ctx context.Context, applicantID int64, category string) ([]entity.DataVersion, error) {
	// For simplicity, returning empty for now or implementing if necessary
	return nil, nil
}

func (r *ApplicantRepo) StoreExtractedField(ctx context.Context, applicantID int64, documentID int64, field, value string) error {
	query := `INSERT INTO extracted_fields (applicant_id, document_id, field_name, field_value) VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(ctx, query, applicantID, documentID, field, value)
	return err
}

func (r *ApplicantRepo) GetLatestDocumentByCategory(ctx context.Context, applicantID int64, category string) (entity.Document, error) {
	// file_type in DB usually matches category (e.g. 'passport', 'diploma')
	query := `SELECT id, applicant_id, file_type, file_name, storage_path, status, uploaded_at FROM applicants_document WHERE applicant_id=$1 AND file_type=$2 ORDER BY uploaded_at DESC LIMIT 1`
	var d entity.Document
	err := r.pool.QueryRow(ctx, query, applicantID, category).Scan(&d.ID, &d.ApplicantID, &d.FileType, &d.FileName, &d.StoragePath, &d.Status, &d.UploadedAt)
	return d, err
}

func (r *ApplicantRepo) StoreEducation(ctx context.Context, data entity.EducationData) error {
	query := `INSERT INTO applicants_data_education (applicant_id, document_id, institution_name, degree_title, major, graduation_date, diploma_serial_number, source, version) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.pool.Exec(ctx, query, data.ApplicantID, data.DocumentID, data.InstitutionName, data.DegreeTitle, data.Major, data.GraduationDate, data.DiplomaSerialNumber, data.Source, data.Version)
	return err
}

func (r *ApplicantRepo) StoreWorkExperience(ctx context.Context, data entity.WorkExperience) error {
	query := `INSERT INTO applicants_data_work_experience (applicant_id, document_id, company_name, position, start_date, end_date, country, city, source, version) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.pool.Exec(ctx, query, data.ApplicantID, data.DocumentID, data.CompanyName, data.Position, data.StartDate, data.EndDate, data.Country, data.City, data.Source, data.Version)
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
			source, version 
		FROM applicants_data_identification 
		WHERE applicant_id=$1 
		ORDER BY version DESC, id DESC 
		LIMIT 1`
	var d entity.IdentificationData
	err := r.pool.QueryRow(ctx, query, applicantID).Scan(
		&d.ID, &d.ApplicantID, &d.DocumentID, &d.Name, &d.Surname, &d.Patronymic, &d.Email, &d.Phone, &d.DocumentNumber, &d.DateOfBirth, &d.Gender, &d.Nationality, &d.Source, &d.Version,
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
	query := `SELECT id, applicant_id, document_id, institution_name, degree_title, major, graduation_date, diploma_serial_number, source, version FROM applicants_data_education WHERE applicant_id=$1 ORDER BY version DESC, id DESC LIMIT 1`
	var d entity.EducationData
	err := r.pool.QueryRow(ctx, query, applicantID).Scan(
		&d.ID, &d.ApplicantID, &d.DocumentID, &d.InstitutionName, &d.DegreeTitle, &d.Major, &d.GraduationDate, &d.DiplomaSerialNumber, &d.Source, &d.Version,
	)
	if err != nil {
		if err.Error() == "no rows in result set" || err.Error() == "pgx: no rows in result set" {
			return d, nil
		}
		return d, err
	}
	return d, nil
}

func (r *ApplicantRepo) ListWorkExperience(ctx context.Context, applicantID int64) ([]entity.WorkExperience, error) {
	query := `SELECT id, applicant_id, document_id, country, city, company_name, position, start_date, end_date, source, version FROM applicants_data_work_experience WHERE applicant_id=$1 ORDER BY start_date DESC`
	rows, err := r.pool.Query(ctx, query, applicantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]entity.WorkExperience, 0)
	for rows.Next() {
		var d entity.WorkExperience
		err = rows.Scan(&d.ID, &d.ApplicantID, &d.DocumentID, &d.Country, &d.City, &d.CompanyName, &d.Position, &d.StartDate, &d.EndDate, &d.Source, &d.Version)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, nil
}

func (r *ApplicantRepo) ListRecommendations(ctx context.Context, applicantID int64) ([]entity.RecommendationData, error) {
	query := `SELECT id, applicant_id, document_id, author_name, author_position, author_institution, key_strengths, source, version FROM applicants_data_recommendation WHERE applicant_id=$1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, applicantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []entity.RecommendationData
	for rows.Next() {
		var d entity.RecommendationData
		err = rows.Scan(&d.ID, &d.ApplicantID, &d.DocumentID, &d.AuthorName, &d.AuthorPosition, &d.AuthorInstitution, &d.KeyStrengths, &d.Source, &d.Version)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, nil
}

func (r *ApplicantRepo) ListAchievements(ctx context.Context, applicantID int64) ([]entity.AchievementData, error) {
	query := `SELECT id, applicant_id, document_id, COALESCE(achievement_title, ''), COALESCE(description, ''), COALESCE(date_received, '1970-01-01'), COALESCE(document_path, ''), COALESCE(achievement_type, ''), COALESCE(company, ''), COALESCE(source, ''), version FROM applicants_data_achievements WHERE applicant_id=$1 ORDER BY date_received DESC`
	rows, err := r.pool.Query(ctx, query, applicantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []entity.AchievementData
	for rows.Next() {
		var d entity.AchievementData
		err = rows.Scan(&d.ID, &d.ApplicantID, &d.DocumentID, &d.AchievementTitle, &d.Description, &d.DateReceived, &d.DocumentPath, &d.AchievementType, &d.Company, &d.Source, &d.Version)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, nil
}

func (r *ApplicantRepo) GetLatestTranscript(ctx context.Context, applicantID int64) (entity.TranscriptData, error) {
	query := `SELECT id, applicant_id, document_id, 
		COALESCE(cumulative_gpa, 0), COALESCE(cumulative_grade, ''), 
		COALESCE(total_credits, 0), COALESCE(obtained_credits, 0), 
		COALESCE(total_semesters, 0), source, version FROM applicants_data_transcript WHERE applicant_id=$1 ORDER BY version DESC, id DESC LIMIT 1`
	var d entity.TranscriptData
	err := r.pool.QueryRow(ctx, query, applicantID).Scan(
		&d.ID, &d.ApplicantID, &d.DocumentID, &d.CumulativeGPA, &d.CumulativeGrade, 
		&d.TotalCredits, &d.ObtainedCredits, &d.TotalSemesters, &d.Source, &d.Version,
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
	query := `SELECT id, applicant_id, document_id, russian_level, english_level, certificate_path, COALESCE(exam_name, ''), COALESCE(score, ''), source, version FROM applicants_data_language_training WHERE applicant_id=$1 ORDER BY version DESC, id DESC LIMIT 1`
	var d entity.LanguageTraining
	err := r.pool.QueryRow(ctx, query, applicantID).Scan(&d.ID, &d.ApplicantID, &d.DocumentID, &d.RussianLevel, &d.EnglishLevel, &d.CertificatePath, &d.ExamName, &d.Score, &d.Source, &d.Version)
	if err != nil {
		if err.Error() == "no rows in result set" || err.Error() == "pgx: no rows in result set" {
			return d, nil
		}
		return d, err
	}
	return d, nil
}

func (r *ApplicantRepo) ListPrograms(ctx context.Context) ([]entity.Program, error) {
	query := `SELECT id, title, year, description, created_at FROM programs ORDER BY year DESC, title ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var programs []entity.Program
	for rows.Next() {
		var p entity.Program
		err = rows.Scan(&p.ID, &p.Title, &p.Year, &p.Description, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		programs = append(programs, p)
	}
	return programs, nil
}

func (r *ApplicantRepo) StoreRecommendation(ctx context.Context, data entity.RecommendationData) error {
	query := `INSERT INTO applicants_data_recommendation (applicant_id, document_id, author_name, author_position, author_institution, key_strengths, source, version) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, query, data.ApplicantID, data.DocumentID, data.AuthorName, data.AuthorPosition, data.AuthorInstitution, data.KeyStrengths, data.Source, data.Version)
	return err
}

func (r *ApplicantRepo) StoreAchievement(ctx context.Context, data entity.AchievementData) error {
	query := `INSERT INTO applicants_data_achievements (applicant_id, document_id, achievement_title, description, date_received, document_path, achievement_type, company, source, version) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.pool.Exec(ctx, query, data.ApplicantID, data.DocumentID, data.AchievementTitle, data.Description, data.DateReceived, data.DocumentPath, data.AchievementType, data.Company, data.Source, data.Version)
	return err
}

func (r *ApplicantRepo) StoreLanguageTraining(ctx context.Context, data entity.LanguageTraining) error {
	query := `INSERT INTO applicants_data_language_training (applicant_id, document_id, russian_level, english_level, certificate_path, exam_name, score, source, version) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.pool.Exec(ctx, query, data.ApplicantID, data.DocumentID, data.RussianLevel, data.EnglishLevel, data.CertificatePath, data.ExamName, data.Score, data.Source, data.Version)
	return err
}

func (r *ApplicantRepo) StoreMotivation(ctx context.Context, data entity.MotivationData) error {
	query := `INSERT INTO applicants_data_motivation (applicant_id, document_id, reasons_for_applying, experience_summary, career_goals, detected_language, main_text, source, version) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.pool.Exec(ctx, query, data.ApplicantID, data.DocumentID, data.ReasonsForApplying, data.ExperienceSummary, data.CareerGoals, data.DetectedLanguage, data.MainText, data.Source, data.Version)
	return err
}

func (r *ApplicantRepo) GetLatestMotivation(ctx context.Context, applicantID int64) (entity.MotivationData, error) {
	query := `SELECT id, applicant_id, document_id, COALESCE(reasons_for_applying, ''), COALESCE(experience_summary, ''), COALESCE(career_goals, ''), COALESCE(detected_language, ''), COALESCE(main_text, ''), source, version FROM applicants_data_motivation WHERE applicant_id=$1 ORDER BY version DESC, id DESC LIMIT 1`
	var d entity.MotivationData
	err := r.pool.QueryRow(ctx, query, applicantID).Scan(&d.ID, &d.ApplicantID, &d.DocumentID, &d.ReasonsForApplying, &d.ExperienceSummary, &d.CareerGoals, &d.DetectedLanguage, &d.MainText, &d.Source, &d.Version)
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
		(applicant_id, document_id, cumulative_gpa, cumulative_grade, total_credits, obtained_credits, total_semesters, source, version) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.pool.Exec(ctx, query, 
		data.ApplicantID, data.DocumentID, data.CumulativeGPA, data.CumulativeGrade, 
		data.TotalCredits, data.ObtainedCredits, data.TotalSemesters, data.Source, data.Version)
	return err
}

func (r *ApplicantRepo) GetTranscriptByDocumentID(ctx context.Context, documentID int64) (entity.TranscriptData, error) {
	query := `SELECT id, applicant_id, document_id, 
		cumulative_gpa, cumulative_grade, total_credits, obtained_credits, 
		total_semesters, source, version FROM applicants_data_transcript WHERE document_id=$1 LIMIT 1`
	var d entity.TranscriptData
	err := r.pool.QueryRow(ctx, query, documentID).Scan(
		&d.ID, &d.ApplicantID, &d.DocumentID, &d.CumulativeGPA, &d.CumulativeGrade, 
		&d.TotalCredits, &d.ObtainedCredits, &d.TotalSemesters, &d.Source, &d.Version,
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
		country=$5, city=$6, source=$7 
		WHERE id=$8`
	_, err := r.pool.Exec(ctx, query,
		data.CompanyName, data.Position, data.StartDate, data.EndDate,
		data.Country, data.City, data.Source, data.ID,
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
		achievement_type=$4, company=$5, source=$6 
		WHERE id=$7`
	_, err := r.pool.Exec(ctx, query,
		data.AchievementTitle, data.Description, data.DateReceived,
		data.AchievementType, data.Company, data.Source, data.ID,
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

func (r *ApplicantRepo) GetProgramByID(ctx context.Context, id int64) (entity.Program, error) {
	query := `SELECT id, title, year, description, created_at FROM programs WHERE id=$1`
	var p entity.Program
	err := r.pool.QueryRow(ctx, query, id).Scan(&p.ID, &p.Title, &p.Year, &p.Description, &p.CreatedAt)
	return p, err
}

func (r *ApplicantRepo) DeleteDataByDocumentID(ctx context.Context, category string, documentID int64) error {
	var query string
	switch category {
	case "passport":
		query = `DELETE FROM applicants_data_identification WHERE document_id=$1`
	case "diploma":
		query = `DELETE FROM applicants_data_education WHERE document_id=$1`
	case "recommendation":
		query = `DELETE FROM applicants_data_recommendation WHERE document_id=$1`
	case "achievement":
		query = `DELETE FROM applicants_data_achievements WHERE document_id=$1`
	case "language":
		query = `DELETE FROM applicants_data_language_training WHERE document_id=$1`
	case "motivation":
		query = `DELETE FROM applicants_data_motivation WHERE document_id=$1`
	case "work":
		query = `DELETE FROM applicants_data_work_experience WHERE document_id=$1`
	case "transcript":
		query = `DELETE FROM applicants_data_transcript WHERE document_id=$1`
	default:
		return nil
	}
	_, err := r.pool.Exec(ctx, query, documentID)
	return err
}

func (r *ApplicantRepo) StoreEvaluation(ctx context.Context, eval entity.ExpertEvaluation) error {
	query := `INSERT INTO expert_evaluations 
		(applicant_id, expert_id, category, score, comment, updated_by_id, is_admin_override, source_info) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, query,
		eval.ApplicantID, eval.ExpertID, eval.Category, eval.Score, eval.Comment,
		eval.UpdatedByID, eval.IsAdminOverride, eval.SourceInfo,
	)
	return err
}

func (r *ApplicantRepo) UpdateEvaluation(ctx context.Context, eval entity.ExpertEvaluation) error {
	query := `UPDATE expert_evaluations SET 
		score=$1, comment=$2, updated_by_id=$3, is_admin_override=$4, source_info=$5, updated_at=NOW() 
		WHERE applicant_id=$6 AND expert_id=$7 AND category=$8`
	_, err := r.pool.Exec(ctx, query,
		eval.Score, eval.Comment, eval.UpdatedByID, eval.IsAdminOverride, eval.SourceInfo,
		eval.ApplicantID, eval.ExpertID, eval.Category,
	)
	return err
}

func (r *ApplicantRepo) ListEvaluations(ctx context.Context, applicantID int64) ([]entity.ExpertEvaluation, error) {
	query := `SELECT id, applicant_id, expert_id, category, score, comment, updated_by_id, is_admin_override, source_info, created_at, updated_at 
		FROM expert_evaluations WHERE applicant_id=$1`
	rows, err := r.pool.Query(ctx, query, applicantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var evals []entity.ExpertEvaluation
	for rows.Next() {
		var e entity.ExpertEvaluation
		err := rows.Scan(&e.ID, &e.ApplicantID, &e.ExpertID, &e.Category, &e.Score, &e.Comment, &e.UpdatedByID, &e.IsAdminOverride, &e.SourceInfo, &e.CreatedAt, &e.UpdatedAt)
		if err != nil {
			return nil, err
		}
		evals = append(evals, e)
	}
	return evals, nil
}

func (r *ApplicantRepo) GetEvaluation(ctx context.Context, applicantID int64, expertID int64, category string) (entity.ExpertEvaluation, error) {
	query := `SELECT id, applicant_id, expert_id, category, score, comment, updated_by_id, is_admin_override, source_info, created_at, updated_at 
		FROM expert_evaluations WHERE applicant_id=$1 AND expert_id=$2 AND category=$3`
	var e entity.ExpertEvaluation
	err := r.pool.QueryRow(ctx, query, applicantID, expertID, category).Scan(
		&e.ID, &e.ApplicantID, &e.ExpertID, &e.Category, &e.Score, &e.Comment, &e.UpdatedByID, &e.IsAdminOverride, &e.SourceInfo, &e.CreatedAt, &e.UpdatedAt,
	)
	return e, err
}

func (r *ApplicantRepo) GetExpertSlots(ctx context.Context) ([]entity.ExpertSlot, error) {
	query := `SELECT user_id, slot_number, created_at FROM expert_slots ORDER BY slot_number ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []entity.ExpertSlot
	for rows.Next() {
		var s entity.ExpertSlot
		err := rows.Scan(&s.UserID, &s.SlotNumber, &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		slots = append(slots, s)
	}
	return slots, nil
}

func (r *ApplicantRepo) AssignExpertSlot(ctx context.Context, userID int64, slotNumber int) error {
	query := `INSERT INTO expert_slots (user_id, slot_number) VALUES ($1, $2) 
		ON CONFLICT (slot_number) DO UPDATE SET user_id = EXCLUDED.user_id`
	_, err := r.pool.Exec(ctx, query, userID, slotNumber)
	return err
}

func (r *ApplicantRepo) RemoveExpertSlot(ctx context.Context, slotNumber int) error {
	query := `DELETE FROM expert_slots WHERE slot_number=$1`
	_, err := r.pool.Exec(ctx, query, slotNumber)
	return err
}

func (r *ApplicantRepo) GetExpertSlotByUserID(ctx context.Context, userID int64) (entity.ExpertSlot, error) {
	query := `SELECT user_id, slot_number, created_at FROM expert_slots WHERE user_id=$1`
	var s entity.ExpertSlot
	err := r.pool.QueryRow(ctx, query, userID).Scan(&s.UserID, &s.SlotNumber, &s.CreatedAt)
	return s, err
}
