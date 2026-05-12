package entity

import "time"

type Program struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Year        int       `json:"year"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type Applicant struct {
	ID              int64      `json:"id"`
	ProgramID       int64      `json:"program_id"`
	Status          string     `json:"status"`
	FirstName       string     `json:"first_name,omitempty"`
	LastName        string     `json:"last_name,omitempty"`
	Patronymic      string     `json:"patronymic,omitempty"`
	Score           float64    `json:"score"` // Maps to aggregated_score for legacy compatibility
	AggregatedScore float64    `json:"aggregated_score"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}

type Document struct {
	ID          int64     `json:"id"`
	ApplicantID int64     `json:"applicant_id"`
	FileType    string    `json:"file_type"`
	FileName    string    `json:"file_name"`
	StoragePath string    `json:"storage_path"`
	Status      string    `json:"status"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

type IdentificationData struct {
	ID             int64     `json:"id"`
	ApplicantID    int64     `json:"applicant_id"`
	DocumentID     *int64    `json:"document_id,omitempty"` // Link to source document
	Email          string    `json:"email"`
	Phone          string    `json:"phone"`
	DocumentNumber string    `json:"document_number"`
	Name           string    `json:"name"`
	Surname        string    `json:"surname"`
	Patronymic     string    `json:"patronymic,omitempty"`
	DateOfBirth    time.Time `json:"date_of_birth"`
	Gender         string    `json:"gender"`
	Nationality    string    `json:"nationality"`
	PhotoPath      string    `json:"photo_path,omitempty"`
	Source         string    `json:"source"`
	CreatedAt      time.Time `json:"created_at"`
}

type EducationData struct {
	ID                  int64     `json:"id"`
	ApplicantID         int64     `json:"applicant_id"`
	DocumentID          *int64    `json:"document_id,omitempty"`
	InstitutionName     string    `json:"institution_name"`
	DegreeTitle         string    `json:"degree_title"`
	Major               string    `json:"major"`
	GraduationDate      time.Time `json:"graduation_date"`
	DiplomaSerialNumber string    `json:"diploma_serial_number"`
	Source              string    `json:"source"`
}

type TranscriptData struct {
	ID              int64     `json:"id"`
	ApplicantID     int64     `json:"applicant_id"`
	DocumentID      *int64    `json:"document_id,omitempty"`
	CumulativeGPA   float64   `json:"cumulative_gpa"`
	CumulativeGrade string    `json:"cumulative_grade"`
	TotalCredits    float64   `json:"total_credits"`
	ObtainedCredits float64   `json:"obtained_credits"`
	TotalSemesters  int       `json:"total_semesters"`
	Source          string    `json:"source"`
	CreatedAt       time.Time `json:"created_at"`
}

type WorkExperience struct {
	ID           int64      `json:"id"`
	ApplicantID  int64      `json:"applicant_id"`
	DocumentID   *int64     `json:"document_id,omitempty"`
	Country      string     `json:"country"`
	City         string     `json:"city"`
	Position     string     `json:"position"`
	CompanyName  string     `json:"company_name"`
	StartDate    time.Time  `json:"start_date"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	RecordType   string     `json:"record_type,omitempty"`
	Competencies string     `json:"competencies,omitempty"`
	Source       string     `json:"source"`
}

type LanguageTraining struct {
	ID              int64  `json:"id"`
	ApplicantID     int64  `json:"applicant_id"`
	DocumentID      *int64 `json:"document_id,omitempty"`
	RussianLevel    string `json:"russian_level"`
	EnglishLevel    string `json:"english_level"`
	ExamName        string `json:"exam_name,omitempty"`
	Score           string `json:"score,omitempty"`
	CertificatePath string `json:"certificate_path,omitempty"`
	Source          string `json:"source"`
}

type MotivationData struct {
	ID                 int64  `json:"id"`
	ApplicantID        int64  `json:"applicant_id"`
	DocumentID         *int64 `json:"document_id,omitempty"`
	ReasonsForApplying string `json:"reasons_for_applying"`
	ExperienceSummary  string `json:"experience_summary"`
	CareerGoals        string `json:"career_goals"`
	DetectedLanguage   string `json:"detected_language"`
	MainText           string `json:"main_text,omitempty"`
	Source             string `json:"source"`
}

type RecommendationData struct {
	ID                int64  `json:"id"`
	ApplicantID       int64  `json:"applicant_id"`
	DocumentID        *int64 `json:"document_id,omitempty"`
	AuthorName        string `json:"author_name"`
	AuthorPosition    string `json:"author_position"`
	AuthorInstitution string `json:"author_institution"`
	KeyStrengths      string `json:"key_strengths"`
	Source            string `json:"source"`
}

type AchievementData struct {
	ID               int64     `json:"id"`
	ApplicantID      int64     `json:"applicant_id"`
	DocumentID       *int64    `json:"document_id,omitempty"`
	AchievementTitle string    `json:"achievement_title"`
	Description      string    `json:"description"`
	DateReceived     time.Time `json:"date_received"`
	AchievementType  string    `json:"achievement_type,omitempty"`
	DocumentPath     string    `json:"document_path,omitempty"`
	Source           string    `json:"source"`
}

type ResumeData struct {
	ID          int64    `json:"id"`
	ApplicantID int64    `json:"applicant_id"`
	DocumentID  *int64   `json:"document_id,omitempty"`
	Summary     string   `json:"summary"`
	Skills      []string `json:"skills"`
	Source      string   `json:"source"`
}

type VideoData struct {
	ID          int64     `json:"id"`
	ApplicantID int64     `json:"applicant_id"`
	VideoURL    string    `json:"video_url"`
	Source      string    `json:"source"`
	CreatedAt   time.Time `json:"created_at"`
}

// DataVersion remains as a generic wrapper if needed for generic history logs
type DataVersion struct {
	ID            int64       `json:"id"`
	ApplicantID   int64       `json:"applicant_id"`
	Category      string      `json:"category"`
	DataContent   interface{} `json:"data_content"`
	VersionNumber int         `json:"version_number"`
	Source        string      `json:"source"`
	AuthorID      int64       `json:"author_id,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
}

// Document processing status constants (8-stage pipeline)
const (
	DocStatusPending              = "pending"               // 1-2: queued, waiting for worker
	DocStatusClassifying          = "classifying"           // 3: AI classification in progress
	DocStatusClassificationFailed = "classification_failed" // 4: AI could not classify — user must pick category
	DocStatusClassified           = "classified"            // 5: category determined, about to extract
	DocStatusExtracting           = "extracting"            // 6: AI data extraction in progress
	DocStatusCompleted            = "completed"             // 7: extracted & saved successfully
	DocStatusExtractionFailed     = "extraction_failed"     // 8: extraction failed — user can enter manually

	// Future scoring statuses (9-11)
	// DocStatusScoring       = "scoring"
	// DocStatusScored        = "scored"
	// DocStatusScoringFailed = "scoring_failed"
)

// Applicant lifecycle status constants
const (
	ApplicantStatusCreated      = "uploaded"
	ApplicantStatusProcessing   = "processing"
	ApplicantStatusVerification = "verifying"
	ApplicantStatusEvaluation   = "assessed"
	ApplicantStatusEvaluated    = "completed"

	EvaluationStatusDraft     = "DRAFT"
	EvaluationStatusCompleted = "COMPLETED"

	CriteriaTypeBase     = "BASE"
	CriteriaTypeBlocking = "BLOCKING"
	CriteriaCodeEnglish  = "ENGLISH_LANG"
)

type ExpertEvaluation struct {
	ID              int64     `json:"id"`
	ApplicantID     int64     `json:"applicant_id"`
	ExpertID        string    `json:"expert_id"`
	Category        string    `json:"category"`
	Score           int       `json:"score"`
	Comment         string    `json:"comment"`
	Status          string    `json:"status"` // DRAFT, COMPLETED
	UpdatedByID     string    `json:"updated_by_id"`
	IsAdminOverride bool      `json:"is_admin_override"`
	IsAIGenerated   bool      `json:"is_ai_generated"`
	SourceInfo      string    `json:"source_info"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type EvaluationCriteria struct {
	Code          string   `json:"code"`
	Title         string   `json:"title"`
	MaxScore      int      `json:"max_score"`
	Type          string   `json:"type"`                 // BASE, BLOCKING
	DocumentTypes []string `json:"document_types"`       // associated document file_type values
	IsMandatory   bool     `json:"is_mandatory"`         // must be uploaded before transfer
	Scheme        string   `json:"scheme"`               // default, ieee
	ProgramID     *int64   `json:"program_id,omitempty"` // nil = applies to all programs
}

type AggregatedEvaluation struct {
	ApplicantID  int64   `json:"applicant_id"`
	AverageScore float64 `json:"average_score"`
	Status       string  `json:"status"`
}

type ExpertSlot struct {
	UserID     string    `json:"user_id"`
	SlotNumber int       `json:"slot_number"`
	ProgramID  int64     `json:"program_id"`
	FirstName  string    `json:"first_name,omitempty"`
	LastName   string    `json:"last_name,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type ScoringResult struct {
	Code    string `json:"code"`
	Score   int    `json:"score"`
	Comment string `json:"comment"`
}

type User struct {
	ID        string   `json:"id"`
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
}

type StagingResult struct {
	StagingID string        `json:"stagingId"`
	Results   []StagingFile `json:"results"`
}

type StagingFile struct {
	StagingFileID string   `json:"stagingFileId"`
	FileName      string   `json:"fileName"`
	Category      string   `json:"category"`
	DocType       string   `json:"docType"`
	Status        string   `json:"status"`
	Warnings      []string `json:"warnings"`
}
