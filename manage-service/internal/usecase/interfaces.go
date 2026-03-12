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

		StoreDataVersion(context.Context, entity.DataVersion) error
		GetDataVersions(context.Context, int64, string) ([]entity.DataVersion, error)
		GetLatestIdentification(context.Context, int64) (entity.IdentificationData, error)
		GetLatestEducation(context.Context, int64) (entity.EducationData, error)
		ListWorkExperience(context.Context, int64) ([]entity.WorkExperience, error)
		ListRecommendations(context.Context, int64) ([]entity.RecommendationData, error)
		ListAchievements(context.Context, int64) ([]entity.AchievementData, error)
		GetLatestTranscript(context.Context, int64) (entity.TranscriptData, error)
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
		ListEvaluations(ctx context.Context, applicantID int64) ([]entity.ExpertEvaluation, error)
		GetEvaluation(ctx context.Context, applicantID int64, expertID int64, category string) (entity.ExpertEvaluation, error)

		GetExpertSlots(ctx context.Context) ([]entity.ExpertSlot, error)
		AssignExpertSlot(ctx context.Context, userID int64, slotNumber int) error
		RemoveExpertSlot(ctx context.Context, slotNumber int) error
		GetExpertSlotByUserID(ctx context.Context, userID int64) (entity.ExpertSlot, error)
	}

	// S3Provider -
	S3Provider interface {
		UploadFile(ctx context.Context, path string, content []byte) error
		GetFile(ctx context.Context, path string) ([]byte, error)
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
)
