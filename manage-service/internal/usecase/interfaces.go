package usecase

import (
	"context"
	"time"

	"manage-service/internal/domain/entity"
)

type (
	ApplicantRepo interface {
		Store(context.Context, *entity.Applicant) error
		Update(context.Context, entity.Applicant) error
		GetByID(context.Context, int64) (entity.Applicant, error)
		List(ctx context.Context, programID int64) ([]entity.Applicant, error)
		Delete(context.Context, int64) error

		StoreIdentification(ctx context.Context, data entity.IdentificationData) error
		StoreEducation(ctx context.Context, data entity.EducationData) error
		StoreRecommendation(ctx context.Context, data entity.RecommendationData) error
		StoreAchievement(ctx context.Context, data entity.AchievementData) error
		StoreLanguageTraining(ctx context.Context, data entity.LanguageTraining) error
		StoreTranscript(ctx context.Context, data entity.TranscriptData) error
		GetTranscriptByDocumentID(ctx context.Context, documentID int64) (entity.TranscriptData, error)

		StoreWorkExperience(ctx context.Context, data entity.WorkExperience) error

		DeleteWorkExperience(context.Context, int64) error
		DeleteRecommendation(context.Context, int64) error
		DeleteAchievement(context.Context, int64) error
		DeleteLanguageTraining(context.Context, int64) error
		DeleteDataByDocumentID(context.Context, string, int64) error

		StoreDataVersion(context.Context, entity.DataVersion) error
		GetDataVersions(context.Context, int64, string) ([]entity.DataVersion, error)
		GetLatestIdentification(context.Context, int64) (entity.IdentificationData, error)
		GetLatestEducation(context.Context, int64) (entity.EducationData, error)
		ListEducation(ctx context.Context, applicantID int64) ([]entity.EducationData, error)
		ListWorkExperience(ctx context.Context, applicantID int64, fileType string) ([]entity.WorkExperience, error)
		ListRecommendations(ctx context.Context, applicantID int64) ([]entity.RecommendationData, error)
		ListAchievements(ctx context.Context, applicantID int64, fileType string) ([]entity.AchievementData, error)
		GetLatestTranscript(ctx context.Context, applicantID int64) (entity.TranscriptData, error)
		GetLatestLanguageTraining(context.Context, int64) (entity.LanguageTraining, error)
		GetLatestMotivation(context.Context, int64) (entity.MotivationData, error)
		GetLatestVideo(context.Context, int64) (entity.VideoData, error)
		UpdateVideo(context.Context, entity.VideoData) error
		StoreMotivation(ctx context.Context, data entity.MotivationData) error

		UpdateIdentification(context.Context, entity.IdentificationData) error
		UpdateEducation(context.Context, entity.EducationData) error
		UpdateTranscript(context.Context, entity.TranscriptData) error
		UpdateWorkExperience(context.Context, entity.WorkExperience) error
		UpdateRecommendation(context.Context, entity.RecommendationData) error
		UpdateAchievement(context.Context, entity.AchievementData) error
		UpdateLanguageTraining(context.Context, entity.LanguageTraining) error
		UpdateMotivation(context.Context, entity.MotivationData) error
		UpdateApplicantRanking(ctx context.Context, applicantID int64, score float64, status string) error
		ConfirmModelData(ctx context.Context, applicantID int64, confirmedBy string) error
		GetScoringScheme(ctx context.Context, applicantID int64) (string, error)
		SetScoringScheme(ctx context.Context, applicantID int64, scheme string) error
	}

	DocumentRepo interface {
		StoreDocument(context.Context, *entity.Document) error
		UpdateDocumentStatus(ctx context.Context, id int64, status string) error
		UpdateDocumentStatusByPath(ctx context.Context, storagePath string, status string) error
		MarkProcessingStarted(ctx context.Context, id int64) error
		GetDocuments(context.Context, int64) ([]entity.Document, error)
		GetDocumentByID(context.Context, int64) (entity.Document, error)
		DeleteDocument(context.Context, int64) error
		UpdateDocumentCategory(ctx context.Context, id int64, category string) error
		UpdateDocumentStoragePath(ctx context.Context, id int64, storagePath string) error
		GetLatestDocumentByCategory(context.Context, int64, string) (entity.Document, error)
		StoreExtractedField(ctx context.Context, applicantID int64, documentID int64, field, value string) error
	}

	ExpertRepo interface {
		StoreEvaluation(ctx context.Context, eval entity.ExpertEvaluation) error
		UpdateEvaluation(ctx context.Context, eval entity.ExpertEvaluation) error
		UpdateEvaluationStatus(ctx context.Context, applicantID int64, expertID string, status string) error
		ListEvaluations(ctx context.Context, applicantID int64) ([]entity.ExpertEvaluation, error)
		GetEvaluation(ctx context.Context, applicantID int64, expertID string, category string) (entity.ExpertEvaluation, error)
		GetCriteria(ctx context.Context) ([]entity.EvaluationCriteria, error)
		CreateCriteria(ctx context.Context, c entity.EvaluationCriteria) error
		UpdateCriteria(ctx context.Context, c entity.EvaluationCriteria) error
		DeleteCriteria(ctx context.Context, code string) error
		SaveEvaluationBatch(ctx context.Context, evaluations []entity.ExpertEvaluation) error
		GetAggregatedScore(ctx context.Context, applicantID int64, categories []string) (float64, error)

		GetExpertSlots(ctx context.Context, programID int64) ([]entity.ExpertSlot, error)
		AssignExpertSlot(ctx context.Context, userID string, slotNumber int, programID int64) error
		RemoveExpertSlot(ctx context.Context, slotNumber int, programID int64) error
		GetExpertSlotByUserID(ctx context.Context, userID string, programID int64) (entity.ExpertSlot, error)
		GetUsersByRoles(ctx context.Context, roles []string) ([]entity.User, error)
	}

	ProgramRepo interface {
		ListPrograms(context.Context) ([]entity.Program, error)
		GetProgramByID(context.Context, int64) (entity.Program, error)
		CreateProgram(ctx context.Context, p entity.Program) (entity.Program, error)
		UpdateProgramStatus(ctx context.Context, id int64, status string) error
	}

	// S3Provider -
	S3Provider interface {
		UploadFile(ctx context.Context, path string, content []byte) error
		GetFile(ctx context.Context, path string) ([]byte, error)
		DeleteFile(ctx context.Context, path string) error
		ListFiles(ctx context.Context, prefix string) ([]string, error)
		CopyFile(ctx context.Context, src, dst string) error
	}

	// ProgramRepo -
	// DocumentQueueRepo -
	DocumentQueueRepo interface {
		Enqueue(context.Context, entity.DocumentQueueTask) (string, error)
		UpdateStatus(ctx context.Context, id string, status string, errMsg *string) error
		GetByApplicantID(context.Context, int64) ([]entity.DocumentQueueTask, error)
		GetStuckTasks(ctx context.Context, olderThan time.Duration) ([]entity.DocumentQueueTask, error)
	}

	DocumentQueueProducer interface {
		PublishTask(task entity.DocumentQueueTask) error
	}

	// ExtractionClient -
	ExtractionClient interface {
		TriggerExtraction(context.Context, entity.Document, []byte) (map[string]string, error)
		ClassifyDocument(ctx context.Context, fileName string, content []byte) (string, []string, error)
		GenerateAnnotation(ctx context.Context, applicantData map[string]interface{}) (string, error)
	}

	Applicant interface {
		CreateApplicant(ctx context.Context, programID int64, firstName, lastName, patronymic string) (entity.Applicant, error)
		GetApplicantData(ctx context.Context, applicantID int64, category string) (interface{}, error)
		ListApplicants(ctx context.Context, programID int64) ([]entity.Applicant, error)
		DeleteApplicant(ctx context.Context, id int64) error
		UpdateApplicantData(ctx context.Context, applicantID int64, category string, rawData map[string]interface{}) error
		DeleteApplicantData(ctx context.Context, applicantID int64, category string, dataID int64) error
		TransferToOperator(ctx context.Context, applicantID int64) error
		TransferToExperts(ctx context.Context, applicantID int64, confirmedBy string) error
	}

	Program interface {
		ListPrograms(context.Context) ([]entity.Program, error)
		GetProgramByID(context.Context, int64) (entity.Program, error)
		CreateProgram(ctx context.Context, p entity.Program) (entity.Program, error)
		UpdateProgramStatus(ctx context.Context, id int64, status string) error
	}

	Document interface {
		UploadDocument(ctx context.Context, applicantID int64, category string, fileName string, content []byte, docType string) (entity.Document, error)
		GetDocuments(ctx context.Context, applicantID int64) ([]entity.Document, error)
		ViewDocument(ctx context.Context, applicantID int64, category string) ([]byte, string, string, error)
		ViewDocumentByID(ctx context.Context, documentID int64) ([]byte, string, string, error)
		DeleteDocument(ctx context.Context, applicantID int64, documentID int64) error
		ReprocessLatestDocument(ctx context.Context, applicantID int64, category string) (int64, error)
		ReprocessDocument(ctx context.Context, documentID int64) error
		ChangeDocumentCategory(ctx context.Context, documentID int64, newCategory string) error
		GetDocumentStatus(ctx context.Context, documentID int64) (string, error)
		UpdateDocumentStatus(ctx context.Context, documentID int64, status string) error
		GetQueueTasks(ctx context.Context, applicantID int64) ([]entity.DocumentQueueTask, error)
		ProcessAIResult(ctx context.Context, applicantID int64, documentID int64, taskCategory string, rawData map[string]string) error
	}

	StagingFileParam struct {
		Name    string
		Content []byte
	}

	Expert interface {
		SaveExpertEvaluation(ctx context.Context, applicantID int64, expertID string, userID string, userName string, role string, evaluations []entity.ExpertEvaluation, complete bool) error
		ListExpertEvaluations(ctx context.Context, applicantID int64, currentUserID string, role string) ([]entity.ExpertEvaluation, error)
		GetEvaluationCriteria(ctx context.Context) ([]entity.EvaluationCriteria, error)
		GetEvaluationCriteriaForApplicant(ctx context.Context, applicantID int64) ([]entity.EvaluationCriteria, string, error)
		GetApplicantScoringScheme(ctx context.Context, applicantID int64) (string, error)
		SetApplicantScoringScheme(ctx context.Context, applicantID int64, scheme string, role string) error
		CreateCriteria(ctx context.Context, c entity.EvaluationCriteria) error
		UpdateCriteria(ctx context.Context, c entity.EvaluationCriteria) error
		DeleteCriteria(ctx context.Context, code string) error
		GetExpertSlots(ctx context.Context, programID int64) ([]entity.ExpertSlot, error)
		AssignExpertSlot(ctx context.Context, userID string, slotNumber int, programID int64, requesterRole string) error
		ListExperts(ctx context.Context) ([]entity.User, error)
	}

	// AIScoringTrigger — узкий интерфейс для запуска AI-оценивания из applicant usecase.
	// Разрывает циклическую зависимость между ApplicantUseCase и ExpertUseCase.
	AIScoringTrigger interface {
		TriggerAIScoring(ctx context.Context, applicantID int64, programID int64)
	}

	// ScoringClient — HTTP-клиент к data-extraction-service для AI-оценивания портфолио.
	ScoringClient interface {
		ScorePortfolio(ctx context.Context, criteria []entity.EvaluationCriteria, applicantData map[string]interface{}) ([]entity.ScoringResult, error)
	}
)
