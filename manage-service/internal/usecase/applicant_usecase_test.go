package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"manage-service/internal/domain/entity"
	"manage-service/internal/usecase"
	"github.com/stretchr/testify/assert"
)

// ─── Mock ApplicantRepo ───────────────────────────────────────────────────────

type mockApplicantRepo struct {
	StoreFunc                   func(context.Context, *entity.Applicant) error
	UpdateFunc                  func(context.Context, entity.Applicant) error
	GetByIDFunc                 func(context.Context, int64) (entity.Applicant, error)
	ListFunc                    func(ctx context.Context, programID int64) ([]entity.Applicant, error)
	DeleteFunc                  func(context.Context, int64) error
	StoreIdentificationFunc     func(ctx context.Context, data entity.IdentificationData) error
	GetLatestIdentificationFunc func(context.Context, int64) (entity.IdentificationData, error)
	GetLatestVideoFunc          func(context.Context, int64) (entity.VideoData, error)
	GetScoringSchemeFunc        func(ctx context.Context, applicantID int64) (string, error)
	UpdateApplicantRankingFunc  func(ctx context.Context, applicantID int64, score float64, status string) error
	ConfirmModelDataFunc        func(ctx context.Context, applicantID int64, confirmedBy string) error
}

// ─── ApplicantRepo stub implementations ──────────────────────────────────────

func (m *mockApplicantRepo) Store(ctx context.Context, a *entity.Applicant) error {
	if m.StoreFunc != nil {
		return m.StoreFunc(ctx, a)
	}
	return nil
}
func (m *mockApplicantRepo) Update(ctx context.Context, a entity.Applicant) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, a)
	}
	return nil
}
func (m *mockApplicantRepo) GetByID(ctx context.Context, id int64) (entity.Applicant, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return entity.Applicant{}, nil
}
func (m *mockApplicantRepo) List(ctx context.Context, programID int64) ([]entity.Applicant, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, programID)
	}
	return nil, nil
}
func (m *mockApplicantRepo) Delete(ctx context.Context, id int64) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}
func (m *mockApplicantRepo) StoreIdentification(ctx context.Context, data entity.IdentificationData) error {
	if m.StoreIdentificationFunc != nil {
		return m.StoreIdentificationFunc(ctx, data)
	}
	return nil
}
func (m *mockApplicantRepo) StoreEducation(ctx context.Context, data entity.EducationData) error {
	return nil
}
func (m *mockApplicantRepo) StoreRecommendation(ctx context.Context, data entity.RecommendationData) error {
	return nil
}
func (m *mockApplicantRepo) StoreAchievement(ctx context.Context, data entity.AchievementData) error {
	return nil
}
func (m *mockApplicantRepo) StoreLanguageTraining(ctx context.Context, data entity.LanguageTraining) error {
	return nil
}
func (m *mockApplicantRepo) StoreTranscript(ctx context.Context, data entity.TranscriptData) error {
	return nil
}
func (m *mockApplicantRepo) GetTranscriptByDocumentID(ctx context.Context, documentID int64) (entity.TranscriptData, error) {
	return entity.TranscriptData{}, nil
}
func (m *mockApplicantRepo) StoreWorkExperience(ctx context.Context, data entity.WorkExperience) error {
	return nil
}
func (m *mockApplicantRepo) DeleteWorkExperience(ctx context.Context, id int64) error { return nil }
func (m *mockApplicantRepo) DeleteRecommendation(ctx context.Context, id int64) error { return nil }
func (m *mockApplicantRepo) DeleteAchievement(ctx context.Context, id int64) error    { return nil }
func (m *mockApplicantRepo) DeleteLanguageTraining(ctx context.Context, id int64) error {
	return nil
}
func (m *mockApplicantRepo) DeleteDataByDocumentID(ctx context.Context, category string, documentID int64) error {
	return nil
}
func (m *mockApplicantRepo) StoreDataVersion(ctx context.Context, v entity.DataVersion) error {
	return nil
}
func (m *mockApplicantRepo) GetDataVersions(ctx context.Context, applicantID int64, category string) ([]entity.DataVersion, error) {
	return nil, nil
}
func (m *mockApplicantRepo) GetLatestIdentification(ctx context.Context, id int64) (entity.IdentificationData, error) {
	if m.GetLatestIdentificationFunc != nil {
		return m.GetLatestIdentificationFunc(ctx, id)
	}
	return entity.IdentificationData{}, nil
}
func (m *mockApplicantRepo) GetLatestEducation(ctx context.Context, id int64) (entity.EducationData, error) {
	return entity.EducationData{}, nil
}
func (m *mockApplicantRepo) ListEducation(ctx context.Context, applicantID int64) ([]entity.EducationData, error) {
	return nil, nil
}
func (m *mockApplicantRepo) ListWorkExperience(ctx context.Context, applicantID int64, fileType string) ([]entity.WorkExperience, error) {
	return nil, nil
}
func (m *mockApplicantRepo) ListRecommendations(ctx context.Context, applicantID int64) ([]entity.RecommendationData, error) {
	return nil, nil
}
func (m *mockApplicantRepo) ListAchievements(ctx context.Context, applicantID int64, fileType string) ([]entity.AchievementData, error) {
	return nil, nil
}
func (m *mockApplicantRepo) GetLatestTranscript(ctx context.Context, id int64) (entity.TranscriptData, error) {
	return entity.TranscriptData{}, nil
}
func (m *mockApplicantRepo) GetLatestLanguageTraining(ctx context.Context, id int64) (entity.LanguageTraining, error) {
	return entity.LanguageTraining{}, nil
}
func (m *mockApplicantRepo) GetLatestMotivation(ctx context.Context, id int64) (entity.MotivationData, error) {
	return entity.MotivationData{}, nil
}
func (m *mockApplicantRepo) GetLatestVideo(ctx context.Context, id int64) (entity.VideoData, error) {
	if m.GetLatestVideoFunc != nil {
		return m.GetLatestVideoFunc(ctx, id)
	}
	return entity.VideoData{}, errors.New("not found")
}
func (m *mockApplicantRepo) UpdateVideo(ctx context.Context, v entity.VideoData) error { return nil }
func (m *mockApplicantRepo) StoreMotivation(ctx context.Context, data entity.MotivationData) error {
	return nil
}
func (m *mockApplicantRepo) UpdateIdentification(ctx context.Context, data entity.IdentificationData) error {
	return nil
}
func (m *mockApplicantRepo) UpdateEducation(ctx context.Context, data entity.EducationData) error {
	return nil
}
func (m *mockApplicantRepo) UpdateTranscript(ctx context.Context, data entity.TranscriptData) error {
	return nil
}
func (m *mockApplicantRepo) UpdateWorkExperience(ctx context.Context, data entity.WorkExperience) error {
	return nil
}
func (m *mockApplicantRepo) UpdateRecommendation(ctx context.Context, data entity.RecommendationData) error {
	return nil
}
func (m *mockApplicantRepo) UpdateAchievement(ctx context.Context, data entity.AchievementData) error {
	return nil
}
func (m *mockApplicantRepo) UpdateLanguageTraining(ctx context.Context, data entity.LanguageTraining) error {
	return nil
}
func (m *mockApplicantRepo) UpdateMotivation(ctx context.Context, data entity.MotivationData) error {
	return nil
}
func (m *mockApplicantRepo) UpdateApplicantRanking(ctx context.Context, applicantID int64, score float64, status string) error {
	if m.UpdateApplicantRankingFunc != nil {
		return m.UpdateApplicantRankingFunc(ctx, applicantID, score, status)
	}
	return nil
}
func (m *mockApplicantRepo) ConfirmModelData(ctx context.Context, applicantID int64, confirmedBy string) error {
	if m.ConfirmModelDataFunc != nil {
		return m.ConfirmModelDataFunc(ctx, applicantID, confirmedBy)
	}
	return nil
}
func (m *mockApplicantRepo) GetScoringScheme(ctx context.Context, applicantID int64) (string, error) {
	if m.GetScoringSchemeFunc != nil {
		return m.GetScoringSchemeFunc(ctx, applicantID)
	}
	return "default", nil
}
func (m *mockApplicantRepo) SetScoringScheme(ctx context.Context, applicantID int64, scheme string) error {
	return nil
}

// ─── Mock DocumentRepo ────────────────────────────────────────────────────────

type mockDocumentRepo struct {
	GetDocumentsFunc func(context.Context, int64) ([]entity.Document, error)
}

func (m *mockDocumentRepo) StoreDocument(ctx context.Context, doc *entity.Document) error { return nil }
func (m *mockDocumentRepo) UpdateDocumentStatus(ctx context.Context, id int64, status string) error {
	return nil
}
func (m *mockDocumentRepo) UpdateDocumentStatusByPath(ctx context.Context, storagePath string, status string) error {
	return nil
}
func (m *mockDocumentRepo) MarkProcessingStarted(ctx context.Context, id int64) error { return nil }
func (m *mockDocumentRepo) GetDocuments(ctx context.Context, applicantID int64) ([]entity.Document, error) {
	if m.GetDocumentsFunc != nil {
		return m.GetDocumentsFunc(ctx, applicantID)
	}
	return nil, nil
}
func (m *mockDocumentRepo) GetDocumentByID(ctx context.Context, id int64) (entity.Document, error) {
	return entity.Document{}, nil
}
func (m *mockDocumentRepo) DeleteDocument(ctx context.Context, id int64) error { return nil }
func (m *mockDocumentRepo) UpdateDocumentCategory(ctx context.Context, id int64, category string) error {
	return nil
}
func (m *mockDocumentRepo) UpdateDocumentStoragePath(ctx context.Context, id int64, storagePath string) error {
	return nil
}
func (m *mockDocumentRepo) GetLatestDocumentByCategory(ctx context.Context, applicantID int64, category string) (entity.Document, error) {
	return entity.Document{}, nil
}
func (m *mockDocumentRepo) StoreExtractedField(ctx context.Context, applicantID int64, documentID int64, field, value string) error {
	return nil
}

// ─── Mock Document UseCase ────────────────────────────────────────────────────

type mockDocumentUC struct{}

func (m *mockDocumentUC) UploadDocument(ctx context.Context, applicantID int64, category string, fileName string, content []byte, docType string) (entity.Document, error) {
	return entity.Document{}, nil
}
func (m *mockDocumentUC) GetDocuments(ctx context.Context, applicantID int64) ([]entity.Document, error) {
	return nil, nil
}
func (m *mockDocumentUC) ViewDocument(ctx context.Context, applicantID int64, category string) ([]byte, string, string, error) {
	return nil, "", "", nil
}
func (m *mockDocumentUC) ViewDocumentByID(ctx context.Context, documentID int64) ([]byte, string, string, error) {
	return nil, "", "", nil
}
func (m *mockDocumentUC) DeleteDocument(ctx context.Context, applicantID int64, documentID int64) error {
	return nil
}
func (m *mockDocumentUC) ReprocessLatestDocument(ctx context.Context, applicantID int64, category string) (int64, error) {
	return 0, nil
}
func (m *mockDocumentUC) ReprocessDocument(ctx context.Context, documentID int64) error { return nil }
func (m *mockDocumentUC) ChangeDocumentCategory(ctx context.Context, documentID int64, newCategory string) error {
	return nil
}
func (m *mockDocumentUC) GetDocumentStatus(ctx context.Context, documentID int64) (string, error) {
	return "", nil
}
func (m *mockDocumentUC) UpdateDocumentStatus(ctx context.Context, documentID int64, status string) error {
	return nil
}
func (m *mockDocumentUC) GetQueueTasks(ctx context.Context, applicantID int64) ([]entity.DocumentQueueTask, error) {
	return nil, nil
}
func (m *mockDocumentUC) ProcessAIResult(ctx context.Context, applicantID int64, documentID int64, taskCategory string, rawData map[string]string) error {
	return nil
}

// ─── Mock ExpertRepo ──────────────────────────────────────────────────────────

type mockExpertRepo struct{}

func (m *mockExpertRepo) StoreEvaluation(ctx context.Context, eval entity.ExpertEvaluation) error {
	return nil
}
func (m *mockExpertRepo) UpdateEvaluation(ctx context.Context, eval entity.ExpertEvaluation) error {
	return nil
}
func (m *mockExpertRepo) UpdateEvaluationStatus(ctx context.Context, applicantID int64, expertID string, status string) error {
	return nil
}
func (m *mockExpertRepo) ListEvaluations(ctx context.Context, applicantID int64) ([]entity.ExpertEvaluation, error) {
	return nil, nil
}
func (m *mockExpertRepo) GetEvaluation(ctx context.Context, applicantID int64, expertID string, category string) (entity.ExpertEvaluation, error) {
	return entity.ExpertEvaluation{}, nil
}
func (m *mockExpertRepo) GetCriteria(ctx context.Context) ([]entity.EvaluationCriteria, error) {
	return nil, nil
}
func (m *mockExpertRepo) CreateCriteria(ctx context.Context, c entity.EvaluationCriteria) error {
	return nil
}
func (m *mockExpertRepo) UpdateCriteria(ctx context.Context, c entity.EvaluationCriteria) error {
	return nil
}
func (m *mockExpertRepo) DeleteCriteria(ctx context.Context, code string) error { return nil }
func (m *mockExpertRepo) SaveEvaluationBatch(ctx context.Context, evaluations []entity.ExpertEvaluation) error {
	return nil
}
func (m *mockExpertRepo) GetAggregatedScore(ctx context.Context, applicantID int64, categories []string) (float64, error) {
	return 0, nil
}
func (m *mockExpertRepo) GetExpertSlots(ctx context.Context, programID int64) ([]entity.ExpertSlot, error) {
	return nil, nil
}
func (m *mockExpertRepo) AssignExpertSlot(ctx context.Context, userID string, slotNumber int, programID int64) error {
	return nil
}
func (m *mockExpertRepo) RemoveExpertSlot(ctx context.Context, slotNumber int, programID int64) error {
	return nil
}
func (m *mockExpertRepo) GetExpertSlotByUserID(ctx context.Context, userID string, programID int64) (entity.ExpertSlot, error) {
	return entity.ExpertSlot{}, nil
}
func (m *mockExpertRepo) GetUsersByRoles(ctx context.Context, roles []string) ([]entity.User, error) {
	return nil, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newApplicantUC(appRepo *mockApplicantRepo, docRepo *mockDocumentRepo) *usecase.ApplicantUseCase {
	return usecase.NewApplicantUseCase(appRepo, docRepo, &mockDocumentUC{}, &mockExpertRepo{})
}

// ─── CreateApplicant ──────────────────────────────────────────────────────────

func TestApplicantUseCase_CreateApplicant_Success(t *testing.T) {
	appRepo := &mockApplicantRepo{
		StoreFunc: func(ctx context.Context, a *entity.Applicant) error {
			a.ID = 42
			return nil
		},
	}
	uc := newApplicantUC(appRepo, &mockDocumentRepo{})
	applicant, err := uc.CreateApplicant(context.Background(), 1, "Иван", "Иванов", "Иванович")
	assert.NoError(t, err)
	assert.Equal(t, int64(42), applicant.ID)
	assert.Equal(t, "uploaded", applicant.Status)
}

func TestApplicantUseCase_CreateApplicant_StoreError(t *testing.T) {
	appRepo := &mockApplicantRepo{
		StoreFunc: func(ctx context.Context, a *entity.Applicant) error {
			return errors.New("db error")
		},
	}
	uc := newApplicantUC(appRepo, &mockDocumentRepo{})
	_, err := uc.CreateApplicant(context.Background(), 1, "Иван", "Иванов", "")
	assert.Error(t, err)
}

func TestApplicantUseCase_CreateApplicant_IdentificationError(t *testing.T) {
	appRepo := &mockApplicantRepo{
		StoreFunc: func(ctx context.Context, a *entity.Applicant) error {
			a.ID = 1
			return nil
		},
		StoreIdentificationFunc: func(ctx context.Context, data entity.IdentificationData) error {
			return errors.New("ident error")
		},
	}
	uc := newApplicantUC(appRepo, &mockDocumentRepo{})
	_, err := uc.CreateApplicant(context.Background(), 1, "Иван", "Иванов", "")
	assert.Error(t, err)
}

// ─── ListApplicants ───────────────────────────────────────────────────────────

func TestApplicantUseCase_ListApplicants(t *testing.T) {
	appRepo := &mockApplicantRepo{
		ListFunc: func(ctx context.Context, programID int64) ([]entity.Applicant, error) {
			return []entity.Applicant{{ID: 1}, {ID: 2}}, nil
		},
	}
	uc := newApplicantUC(appRepo, &mockDocumentRepo{})
	list, err := uc.ListApplicants(context.Background(), 0)
	assert.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestApplicantUseCase_ListApplicants_Error(t *testing.T) {
	appRepo := &mockApplicantRepo{
		ListFunc: func(ctx context.Context, programID int64) ([]entity.Applicant, error) {
			return nil, errors.New("db error")
		},
	}
	uc := newApplicantUC(appRepo, &mockDocumentRepo{})
	_, err := uc.ListApplicants(context.Background(), 0)
	assert.Error(t, err)
}

// ─── DeleteApplicant ──────────────────────────────────────────────────────────

func TestApplicantUseCase_DeleteApplicant_Success(t *testing.T) {
	deleted := false
	appRepo := &mockApplicantRepo{
		DeleteFunc: func(ctx context.Context, id int64) error {
			deleted = true
			return nil
		},
	}
	docRepo := &mockDocumentRepo{
		GetDocumentsFunc: func(ctx context.Context, id int64) ([]entity.Document, error) {
			return []entity.Document{{ID: 10, ApplicantID: id}}, nil
		},
	}
	uc := newApplicantUC(appRepo, docRepo)
	err := uc.DeleteApplicant(context.Background(), 1)
	assert.NoError(t, err)
	assert.True(t, deleted)
}

// ─── DeleteApplicantData ──────────────────────────────────────────────────────

func TestApplicantUseCase_DeleteApplicantData_UnknownCategory(t *testing.T) {
	uc := newApplicantUC(&mockApplicantRepo{}, &mockDocumentRepo{})
	err := uc.DeleteApplicantData(context.Background(), 1, "unknown_cat", 10)
	assert.Error(t, err)
}

func TestApplicantUseCase_DeleteApplicantData_WorkCategory(t *testing.T) {
	uc := newApplicantUC(&mockApplicantRepo{}, &mockDocumentRepo{})
	err := uc.DeleteApplicantData(context.Background(), 1, "work", 10)
	assert.NoError(t, err)
}

// ─── GetApplicantData ─────────────────────────────────────────────────────────

func TestApplicantUseCase_GetApplicantData_UnknownCategory(t *testing.T) {
	uc := newApplicantUC(&mockApplicantRepo{}, &mockDocumentRepo{})
	_, err := uc.GetApplicantData(context.Background(), 1, "nonexistent")
	assert.Error(t, err)
}

func TestApplicantUseCase_GetApplicantData_Passport(t *testing.T) {
	appRepo := &mockApplicantRepo{
		GetLatestIdentificationFunc: func(ctx context.Context, id int64) (entity.IdentificationData, error) {
			return entity.IdentificationData{ApplicantID: id, Name: "Иван"}, nil
		},
	}
	uc := newApplicantUC(appRepo, &mockDocumentRepo{})
	data, err := uc.GetApplicantData(context.Background(), 1, "passport")
	assert.NoError(t, err)
	assert.NotNil(t, data)
}

// ─── TransferToExperts ────────────────────────────────────────────────────────

func TestApplicantUseCase_TransferToExperts_Success(t *testing.T) {
	rankingUpdated := false
	appRepo := &mockApplicantRepo{
		UpdateApplicantRankingFunc: func(ctx context.Context, applicantID int64, score float64, status string) error {
			rankingUpdated = true
			assert.Equal(t, entity.ApplicantStatusEvaluation, status)
			return nil
		},
	}
	uc := newApplicantUC(appRepo, &mockDocumentRepo{})
	err := uc.TransferToExperts(context.Background(), 1, "")
	assert.NoError(t, err)
	assert.True(t, rankingUpdated)
}

// ensure time package is referenced (used in entity fields)
var _ = time.Time{}
