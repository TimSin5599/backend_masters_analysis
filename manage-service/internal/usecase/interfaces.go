package usecase

import (
	"context"
	"manage-service/internal/entity"
)

type (
	// ApplicantRepo -
	ApplicantRepo interface {
		Store(context.Context, *entity.Applicant) error
		Update(context.Context, entity.Applicant) error
		GetByID(context.Context, int64) (entity.Applicant, error)
		List(ctx context.Context, programID int64) ([]entity.Applicant, error)
		Delete(context.Context, int64) error
		ListPrograms(context.Context) ([]entity.Program, error)
		GetProgramByID(context.Context, int64) (entity.Program, error)

		StoreDocument(context.Context, *entity.Document) error
		UpdateDocumentStatus(ctx context.Context, id int64, status string) error
		GetDocuments(context.Context, int64) ([]entity.Document, error)
		GetDocumentByID(context.Context, int64) (entity.Document, error)

		StoreExtractedField(ctx context.Context, applicantID int64, documentID int64, field, value string) error
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
		DeleteDocument(context.Context, int64) error

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
		GetLatestDocumentByCategory(context.Context, int64, string) (entity.Document, error)
		StoreMotivation(ctx context.Context, data entity.MotivationData) error

		UpdateIdentification(context.Context, entity.IdentificationData) error
		UpdateEducation(context.Context, entity.EducationData) error
		UpdateTranscript(context.Context, entity.TranscriptData) error
		UpdateWorkExperience(context.Context, entity.WorkExperience) error
		UpdateRecommendation(context.Context, entity.RecommendationData) error
		UpdateAchievement(context.Context, entity.AchievementData) error
		UpdateLanguageTraining(context.Context, entity.LanguageTraining) error
		UpdateMotivation(context.Context, entity.MotivationData) error

		StoreEvaluation(ctx context.Context, eval entity.ExpertEvaluation) error
		UpdateEvaluation(ctx context.Context, eval entity.ExpertEvaluation) error
		UpdateEvaluationStatus(ctx context.Context, applicantID int64, expertID string, status string) error
		ListEvaluations(ctx context.Context, applicantID int64) ([]entity.ExpertEvaluation, error)
		GetEvaluation(ctx context.Context, applicantID int64, expertID string, category string) (entity.ExpertEvaluation, error)
		GetCriteria(ctx context.Context) ([]entity.EvaluationCriteria, error)
		SaveEvaluationBatch(ctx context.Context, evaluations []entity.ExpertEvaluation) error
		GetAggregatedScore(ctx context.Context, applicantID int64) (float64, error)
		UpdateApplicantRanking(ctx context.Context, applicantID int64, score float64, status string) error

		GetExpertSlots(ctx context.Context) ([]entity.ExpertSlot, error)
		AssignExpertSlot(ctx context.Context, userID string, slotNumber int) error
		RemoveExpertSlot(ctx context.Context, slotNumber int) error
		GetExpertSlotByUserID(ctx context.Context, userID string) (entity.ExpertSlot, error)
		GetUsersByRoles(ctx context.Context, roles []string) ([]entity.User, error)
	}

	// S3Provider -
	S3Provider interface {
		UploadFile(ctx context.Context, path string, content []byte) error
		GetFile(ctx context.Context, path string) ([]byte, error)
		DeleteFile(ctx context.Context, path string) error
	}

	// ProgramRepo -
	ProgramRepo interface {
		Store(context.Context, entity.Program) error
		List(context.Context) ([]entity.Program, error)
	}

	// DocumentQueueRepo -
	DocumentQueueRepo interface {
		Enqueue(context.Context, entity.DocumentQueueTask) (string, error)
		UpdateStatus(ctx context.Context, id string, status string, errMsg *string) error
		GetByApplicantID(context.Context, int64) ([]entity.DocumentQueueTask, error)
	}

	DocumentQueueProducer interface {
		PublishTask(task entity.DocumentQueueTask) error
	}

	// ExtractionClient -
	ExtractionClient interface {
		TriggerExtraction(context.Context, entity.Document, []byte) (map[string]string, error)
	}

	UseCase interface {
		Store(context.Context, *entity.Applicant) (int64, error)
		Update(context.Context, entity.Applicant) error
		GetByID(context.Context, int64) (entity.Applicant, error)
		List(ctx context.Context, programID int64) ([]entity.Applicant, error)
		Delete(context.Context, int64) error

		ListPrograms(context.Context) ([]entity.Program, error)
		GetProgramByID(context.Context, int64) (entity.Program, error)

		StoreDocument(context.Context, *entity.Document) (int64, error)
		UpdateDocumentStatus(ctx context.Context, id int64, status string) error
		GetDocuments(context.Context, int64) ([]entity.Document, error)
		GetDocumentByID(context.Context, int64) (entity.Document, error)
		GetFileContent(ctx context.Context, documentID int64) ([]byte, string, string, error)
		GetDocumentStatus(ctx context.Context, documentID int64) (string, error)

		SaveExpertEvaluation(ctx context.Context, applicantID int64, expertID string, userID string, userName string, role string, evaluations []entity.ExpertEvaluation, complete bool) error
		ListExpertEvaluations(ctx context.Context, applicantID int64, currentUserID string) ([]entity.ExpertEvaluation, error)
		GetEvaluationCriteria(ctx context.Context) ([]entity.EvaluationCriteria, error)

		GetExpertSlots(ctx context.Context) ([]entity.ExpertSlot, error)
		AssignExpertSlot(ctx context.Context, userID string, slotNumber int, requesterRole string) error
		ListExperts(ctx context.Context) ([]entity.User, error)
	}
)
