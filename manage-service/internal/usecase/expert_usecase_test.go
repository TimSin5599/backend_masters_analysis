package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"manage-service/internal/domain"
	"manage-service/internal/domain/entity"
	"manage-service/internal/usecase"
)

// ─── Mock ExpertRepo (with function fields) ───────────────────────────────────

type mockExpertRepoFull struct {
	GetCriteriaFunc           func(ctx context.Context) ([]entity.EvaluationCriteria, error)
	ListEvaluationsFunc       func(ctx context.Context, applicantID int64) ([]entity.ExpertEvaluation, error)
	GetExpertSlotByUserIDFunc func(ctx context.Context, userID string, programID int64) (entity.ExpertSlot, error)
	SaveEvaluationBatchFunc   func(ctx context.Context, evals []entity.ExpertEvaluation) error
	GetExpertSlotsFunc        func(ctx context.Context, programID int64) ([]entity.ExpertSlot, error)
	GetUsersByRolesFunc       func(ctx context.Context, roles []string) ([]entity.User, error)
	GetAggregatedScoreFunc    func(ctx context.Context, applicantID int64, categories []string) (float64, error)
}

func (m *mockExpertRepoFull) StoreEvaluation(ctx context.Context, eval entity.ExpertEvaluation) error {
	return nil
}
func (m *mockExpertRepoFull) UpdateEvaluation(ctx context.Context, eval entity.ExpertEvaluation) error {
	return nil
}
func (m *mockExpertRepoFull) UpdateEvaluationStatus(ctx context.Context, applicantID int64, expertID string, status string) error {
	return nil
}
func (m *mockExpertRepoFull) ListEvaluations(ctx context.Context, applicantID int64) ([]entity.ExpertEvaluation, error) {
	if m.ListEvaluationsFunc != nil {
		return m.ListEvaluationsFunc(ctx, applicantID)
	}
	return nil, nil
}
func (m *mockExpertRepoFull) GetEvaluation(ctx context.Context, applicantID int64, expertID string, category string) (entity.ExpertEvaluation, error) {
	return entity.ExpertEvaluation{}, errors.New("not found")
}
func (m *mockExpertRepoFull) GetCriteria(ctx context.Context) ([]entity.EvaluationCriteria, error) {
	if m.GetCriteriaFunc != nil {
		return m.GetCriteriaFunc(ctx)
	}
	return nil, nil
}
func (m *mockExpertRepoFull) CreateCriteria(ctx context.Context, c entity.EvaluationCriteria) error {
	return nil
}
func (m *mockExpertRepoFull) UpdateCriteria(ctx context.Context, c entity.EvaluationCriteria) error {
	return nil
}
func (m *mockExpertRepoFull) DeleteCriteria(ctx context.Context, code string) error { return nil }
func (m *mockExpertRepoFull) SaveEvaluationBatch(ctx context.Context, evals []entity.ExpertEvaluation) error {
	if m.SaveEvaluationBatchFunc != nil {
		return m.SaveEvaluationBatchFunc(ctx, evals)
	}
	return nil
}
func (m *mockExpertRepoFull) GetAggregatedScore(ctx context.Context, applicantID int64, categories []string) (float64, error) {
	if m.GetAggregatedScoreFunc != nil {
		return m.GetAggregatedScoreFunc(ctx, applicantID, categories)
	}
	return 0, nil
}
func (m *mockExpertRepoFull) GetExpertSlots(ctx context.Context, programID int64) ([]entity.ExpertSlot, error) {
	if m.GetExpertSlotsFunc != nil {
		return m.GetExpertSlotsFunc(ctx, programID)
	}
	return nil, nil
}
func (m *mockExpertRepoFull) AssignExpertSlot(ctx context.Context, userID string, slotNumber int, programID int64) error {
	return nil
}
func (m *mockExpertRepoFull) RemoveExpertSlot(ctx context.Context, slotNumber int, programID int64) error {
	return nil
}
func (m *mockExpertRepoFull) GetExpertSlotByUserID(ctx context.Context, userID string, programID int64) (entity.ExpertSlot, error) {
	if m.GetExpertSlotByUserIDFunc != nil {
		return m.GetExpertSlotByUserIDFunc(ctx, userID, programID)
	}
	return entity.ExpertSlot{UserID: userID}, nil
}
func (m *mockExpertRepoFull) GetUsersByRoles(ctx context.Context, roles []string) ([]entity.User, error) {
	if m.GetUsersByRolesFunc != nil {
		return m.GetUsersByRolesFunc(ctx, roles)
	}
	return nil, nil
}

// ─── Mock ApplicantRepo (minimal for expert usecase) ─────────────────────────

type mockAppRepoForExpert struct {
	GetByIDFunc                func(ctx context.Context, id int64) (entity.Applicant, error)
	GetScoringSchemeFunc       func(ctx context.Context, applicantID int64) (string, error)
	SetScoringSchemeFunc       func(ctx context.Context, applicantID int64, scheme string) error
	UpdateApplicantRankingFunc func(ctx context.Context, applicantID int64, score float64, status string) error
}

func (m *mockAppRepoForExpert) Store(ctx context.Context, a *entity.Applicant) error { return nil }
func (m *mockAppRepoForExpert) Update(ctx context.Context, a entity.Applicant) error { return nil }
func (m *mockAppRepoForExpert) GetByID(ctx context.Context, id int64) (entity.Applicant, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return entity.Applicant{ID: id, ProgramID: 1}, nil
}
func (m *mockAppRepoForExpert) List(ctx context.Context, programID int64) ([]entity.Applicant, error) {
	return nil, nil
}
func (m *mockAppRepoForExpert) Delete(ctx context.Context, id int64) error { return nil }
func (m *mockAppRepoForExpert) StoreIdentification(ctx context.Context, data entity.IdentificationData) error {
	return nil
}
func (m *mockAppRepoForExpert) StoreEducation(ctx context.Context, data entity.EducationData) error {
	return nil
}
func (m *mockAppRepoForExpert) StoreRecommendation(ctx context.Context, data entity.RecommendationData) error {
	return nil
}
func (m *mockAppRepoForExpert) StoreAchievement(ctx context.Context, data entity.AchievementData) error {
	return nil
}
func (m *mockAppRepoForExpert) StoreLanguageTraining(ctx context.Context, data entity.LanguageTraining) error {
	return nil
}
func (m *mockAppRepoForExpert) StoreTranscript(ctx context.Context, data entity.TranscriptData) error {
	return nil
}
func (m *mockAppRepoForExpert) GetTranscriptByDocumentID(ctx context.Context, documentID int64) (entity.TranscriptData, error) {
	return entity.TranscriptData{}, nil
}
func (m *mockAppRepoForExpert) StoreWorkExperience(ctx context.Context, data entity.WorkExperience) error {
	return nil
}
func (m *mockAppRepoForExpert) DeleteWorkExperience(ctx context.Context, id int64) error { return nil }
func (m *mockAppRepoForExpert) DeleteRecommendation(ctx context.Context, id int64) error { return nil }
func (m *mockAppRepoForExpert) DeleteAchievement(ctx context.Context, id int64) error    { return nil }
func (m *mockAppRepoForExpert) DeleteLanguageTraining(ctx context.Context, id int64) error {
	return nil
}
func (m *mockAppRepoForExpert) DeleteDataByDocumentID(ctx context.Context, category string, documentID int64) error {
	return nil
}
func (m *mockAppRepoForExpert) StoreDataVersion(ctx context.Context, v entity.DataVersion) error {
	return nil
}
func (m *mockAppRepoForExpert) GetDataVersions(ctx context.Context, applicantID int64, category string) ([]entity.DataVersion, error) {
	return nil, nil
}
func (m *mockAppRepoForExpert) GetLatestIdentification(ctx context.Context, id int64) (entity.IdentificationData, error) {
	return entity.IdentificationData{}, nil
}
func (m *mockAppRepoForExpert) GetLatestEducation(ctx context.Context, id int64) (entity.EducationData, error) {
	return entity.EducationData{}, nil
}
func (m *mockAppRepoForExpert) ListEducation(ctx context.Context, applicantID int64) ([]entity.EducationData, error) {
	return nil, nil
}
func (m *mockAppRepoForExpert) ListWorkExperience(ctx context.Context, applicantID int64, fileType string) ([]entity.WorkExperience, error) {
	return nil, nil
}
func (m *mockAppRepoForExpert) ListRecommendations(ctx context.Context, applicantID int64) ([]entity.RecommendationData, error) {
	return nil, nil
}
func (m *mockAppRepoForExpert) ListAchievements(ctx context.Context, applicantID int64, fileType string) ([]entity.AchievementData, error) {
	return nil, nil
}
func (m *mockAppRepoForExpert) GetLatestTranscript(ctx context.Context, id int64) (entity.TranscriptData, error) {
	return entity.TranscriptData{}, nil
}
func (m *mockAppRepoForExpert) GetLatestLanguageTraining(ctx context.Context, id int64) (entity.LanguageTraining, error) {
	return entity.LanguageTraining{}, nil
}
func (m *mockAppRepoForExpert) GetLatestMotivation(ctx context.Context, id int64) (entity.MotivationData, error) {
	return entity.MotivationData{}, nil
}
func (m *mockAppRepoForExpert) GetLatestVideo(ctx context.Context, id int64) (entity.VideoData, error) {
	return entity.VideoData{}, nil
}
func (m *mockAppRepoForExpert) UpdateVideo(ctx context.Context, v entity.VideoData) error { return nil }
func (m *mockAppRepoForExpert) StoreMotivation(ctx context.Context, data entity.MotivationData) error {
	return nil
}
func (m *mockAppRepoForExpert) UpdateIdentification(ctx context.Context, data entity.IdentificationData) error {
	return nil
}
func (m *mockAppRepoForExpert) UpdateEducation(ctx context.Context, data entity.EducationData) error {
	return nil
}
func (m *mockAppRepoForExpert) UpdateTranscript(ctx context.Context, data entity.TranscriptData) error {
	return nil
}
func (m *mockAppRepoForExpert) UpdateWorkExperience(ctx context.Context, data entity.WorkExperience) error {
	return nil
}
func (m *mockAppRepoForExpert) UpdateRecommendation(ctx context.Context, data entity.RecommendationData) error {
	return nil
}
func (m *mockAppRepoForExpert) UpdateAchievement(ctx context.Context, data entity.AchievementData) error {
	return nil
}
func (m *mockAppRepoForExpert) UpdateLanguageTraining(ctx context.Context, data entity.LanguageTraining) error {
	return nil
}
func (m *mockAppRepoForExpert) UpdateMotivation(ctx context.Context, data entity.MotivationData) error {
	return nil
}
func (m *mockAppRepoForExpert) UpdateApplicantRanking(ctx context.Context, applicantID int64, score float64, status string) error {
	if m.UpdateApplicantRankingFunc != nil {
		return m.UpdateApplicantRankingFunc(ctx, applicantID, score, status)
	}
	return nil
}
func (m *mockAppRepoForExpert) ConfirmModelData(ctx context.Context, applicantID int64, confirmedBy string) error {
	return nil
}
func (m *mockAppRepoForExpert) GetScoringScheme(ctx context.Context, applicantID int64) (string, error) {
	if m.GetScoringSchemeFunc != nil {
		return m.GetScoringSchemeFunc(ctx, applicantID)
	}
	return "default", nil
}
func (m *mockAppRepoForExpert) SetScoringScheme(ctx context.Context, applicantID int64, scheme string) error {
	if m.SetScoringSchemeFunc != nil {
		return m.SetScoringSchemeFunc(ctx, applicantID, scheme)
	}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newExpertUC(expertRepo *mockExpertRepoFull, appRepo *mockAppRepoForExpert) *usecase.ExpertUseCase {
	return usecase.NewExpertUseCase(expertRepo, appRepo, nil)
}

// ─── GetEvaluationCriteria ────────────────────────────────────────────────────

func TestExpertUseCase_GetEvaluationCriteria(t *testing.T) {
	expertRepo := &mockExpertRepoFull{
		GetCriteriaFunc: func(ctx context.Context) ([]entity.EvaluationCriteria, error) {
			return []entity.EvaluationCriteria{
				{Code: "C1", Title: "Критерий 1", MaxScore: 10},
			}, nil
		},
	}
	uc := newExpertUC(expertRepo, &mockAppRepoForExpert{})
	criteria, err := uc.GetEvaluationCriteria(context.Background())
	assert.NoError(t, err)
	assert.Len(t, criteria, 1)
	assert.Equal(t, "C1", criteria[0].Code)
}

// ─── SaveExpertEvaluation — immutability check ────────────────────────────────

func TestExpertUseCase_SaveExpertEvaluation_ImmutableCompleted(t *testing.T) {
	expertRepo := &mockExpertRepoFull{
		GetExpertSlotByUserIDFunc: func(ctx context.Context, userID string, programID int64) (entity.ExpertSlot, error) {
			return entity.ExpertSlot{UserID: userID}, nil
		},
		ListEvaluationsFunc: func(ctx context.Context, applicantID int64) ([]entity.ExpertEvaluation, error) {
			return []entity.ExpertEvaluation{
				{ExpertID: "expert1", Status: entity.EvaluationStatusCompleted},
			}, nil
		},
		GetCriteriaFunc: func(ctx context.Context) ([]entity.EvaluationCriteria, error) {
			return []entity.EvaluationCriteria{{Code: "C1", MaxScore: 10}}, nil
		},
	}
	uc := newExpertUC(expertRepo, &mockAppRepoForExpert{})
	scores := []entity.ExpertEvaluation{{Category: "C1", Score: 5}}
	err := uc.SaveExpertEvaluation(context.Background(), 1, "expert1", "expert1", "Эксперт", "expert", scores, false)
	assert.ErrorIs(t, err, domain.ErrEvaluationImmutable)
}

// ─── SaveExpertEvaluation — score exceeds max ─────────────────────────────────

func TestExpertUseCase_SaveExpertEvaluation_ScoreExceedsMax(t *testing.T) {
	expertRepo := &mockExpertRepoFull{
		GetExpertSlotByUserIDFunc: func(ctx context.Context, userID string, programID int64) (entity.ExpertSlot, error) {
			return entity.ExpertSlot{UserID: userID}, nil
		},
		ListEvaluationsFunc: func(ctx context.Context, applicantID int64) ([]entity.ExpertEvaluation, error) {
			return nil, nil
		},
		GetCriteriaFunc: func(ctx context.Context) ([]entity.EvaluationCriteria, error) {
			return []entity.EvaluationCriteria{{Code: "C1", MaxScore: 5}}, nil
		},
	}
	uc := newExpertUC(expertRepo, &mockAppRepoForExpert{})
	scores := []entity.ExpertEvaluation{{Category: "C1", Score: 999}}
	err := uc.SaveExpertEvaluation(context.Background(), 1, "expert1", "expert1", "Эксперт", "expert", scores, false)
	assert.ErrorIs(t, err, domain.ErrScoreExceedsMax)
}

// ─── SaveExpertEvaluation — success (draft) ───────────────────────────────────

func TestExpertUseCase_SaveExpertEvaluation_Success(t *testing.T) {
	saved := false
	expertRepo := &mockExpertRepoFull{
		GetExpertSlotByUserIDFunc: func(ctx context.Context, userID string, programID int64) (entity.ExpertSlot, error) {
			return entity.ExpertSlot{UserID: userID}, nil
		},
		ListEvaluationsFunc: func(ctx context.Context, applicantID int64) ([]entity.ExpertEvaluation, error) {
			return nil, nil
		},
		GetCriteriaFunc: func(ctx context.Context) ([]entity.EvaluationCriteria, error) {
			return []entity.EvaluationCriteria{{Code: "C1", MaxScore: 10}}, nil
		},
		SaveEvaluationBatchFunc: func(ctx context.Context, evals []entity.ExpertEvaluation) error {
			saved = true
			return nil
		},
	}
	uc := newExpertUC(expertRepo, &mockAppRepoForExpert{})
	scores := []entity.ExpertEvaluation{{Category: "C1", Score: 7}}
	err := uc.SaveExpertEvaluation(context.Background(), 1, "expert1", "expert1", "Эксперт", "expert", scores, false)
	assert.NoError(t, err)
	assert.True(t, saved)
}

// ─── SaveExpertEvaluation — admin can override completed evaluations ───────────

func TestExpertUseCase_SaveExpertEvaluation_AdminOverride(t *testing.T) {
	expertRepo := &mockExpertRepoFull{
		ListEvaluationsFunc: func(ctx context.Context, applicantID int64) ([]entity.ExpertEvaluation, error) {
			return []entity.ExpertEvaluation{
				{ExpertID: "expert1", Status: entity.EvaluationStatusCompleted},
			}, nil
		},
		GetCriteriaFunc: func(ctx context.Context) ([]entity.EvaluationCriteria, error) {
			return []entity.EvaluationCriteria{{Code: "C1", MaxScore: 10}}, nil
		},
	}
	uc := newExpertUC(expertRepo, &mockAppRepoForExpert{})
	scores := []entity.ExpertEvaluation{{Category: "C1", Score: 8}}
	// Admin role — skips slot check and immutability guard
	err := uc.SaveExpertEvaluation(context.Background(), 1, "expert1", "admin-uid", "Admin", "admin", scores, false)
	assert.NoError(t, err)
}

// ─── ListExpertEvaluations ────────────────────────────────────────────────────

func TestExpertUseCase_ListExpertEvaluations_ExpertSeesHiddenScores(t *testing.T) {
	expertRepo := &mockExpertRepoFull{
		ListEvaluationsFunc: func(ctx context.Context, applicantID int64) ([]entity.ExpertEvaluation, error) {
			return []entity.ExpertEvaluation{
				{ExpertID: "e1", Score: 5, Status: entity.EvaluationStatusDraft},
				{ExpertID: "e2", Score: 7, Status: entity.EvaluationStatusDraft},
			}, nil
		},
	}
	uc := newExpertUC(expertRepo, &mockAppRepoForExpert{})
	// e1 hasn't completed their evaluation yet — blind grading hides e2's score
	evals, err := uc.ListExpertEvaluations(context.Background(), 1, "e1", "expert")
	assert.NoError(t, err)
	assert.Len(t, evals, 2)
	// Own score is visible
	var e1Eval, e2Eval entity.ExpertEvaluation
	for _, ev := range evals {
		if ev.ExpertID == "e1" {
			e1Eval = ev
		} else {
			e2Eval = ev
		}
	}
	assert.Equal(t, 5, e1Eval.Score)
	// Other expert's score is hidden (-1) until e1 completes
	assert.Equal(t, -1, e2Eval.Score)
}

func TestExpertUseCase_ListExpertEvaluations_AdminSeesAll(t *testing.T) {
	expertRepo := &mockExpertRepoFull{
		ListEvaluationsFunc: func(ctx context.Context, applicantID int64) ([]entity.ExpertEvaluation, error) {
			return []entity.ExpertEvaluation{
				{ExpertID: "e1", Score: 5},
				{ExpertID: "e2", Score: 7},
			}, nil
		},
	}
	uc := newExpertUC(expertRepo, &mockAppRepoForExpert{})
	evals, err := uc.ListExpertEvaluations(context.Background(), 1, "admin-uid", "admin")
	assert.NoError(t, err)
	assert.Len(t, evals, 2)
}
