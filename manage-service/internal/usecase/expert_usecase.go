package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"manage-service/internal/domain"
	"manage-service/internal/domain/entity"
)

type ExpertUseCase struct {
	repo          ExpertRepo
	appRepo       ApplicantRepo
	scoringClient ScoringClient // HTTP-клиент к data-extraction-service /v1/score
}

func NewExpertUseCase(repo ExpertRepo, appRepo ApplicantRepo, scoringClient ScoringClient) *ExpertUseCase {
	return &ExpertUseCase{repo: repo, appRepo: appRepo, scoringClient: scoringClient}
}

func (uc *ExpertUseCase) GetEvaluationCriteria(ctx context.Context) ([]entity.EvaluationCriteria, error) {
	return uc.repo.GetCriteria(ctx)
}

func (uc *ExpertUseCase) CreateCriteria(ctx context.Context, c entity.EvaluationCriteria) error {
	return uc.repo.CreateCriteria(ctx, c)
}

func (uc *ExpertUseCase) UpdateCriteria(ctx context.Context, c entity.EvaluationCriteria) error {
	return uc.repo.UpdateCriteria(ctx, c)
}

func (uc *ExpertUseCase) DeleteCriteria(ctx context.Context, code string) error {
	return uc.repo.DeleteCriteria(ctx, code)
}

func (uc *ExpertUseCase) SaveExpertEvaluation(ctx context.Context, applicantID int64, expertID string, userID string, userName string, role string, evaluations []entity.ExpertEvaluation, complete bool) error {
	isAdmin := role == "admin"
	isAISystem := expertID == "AI_SYSTEM"

	// 1. Получаем programID абитуриента (нужен для проверки слотов per-программе)
	applicant, err := uc.appRepo.GetByID(ctx, applicantID)
	if err != nil {
		return fmt.Errorf("failed to fetch applicant: %w", err)
	}
	programID := applicant.ProgramID

	// 2. Эксперт может сохранять только свои оценки
	if !isAdmin && !isAISystem && expertID != userID {
		return fmt.Errorf("experts can only submit their own evaluations")
	}

	// 3. Проверяем, что целевой expertID является назначенным экспертом для этой программы
	//    AI_SYSTEM и admin пропускают эту проверку
	if !isAISystem && !isAdmin {
		_, err := uc.repo.GetExpertSlotByUserID(ctx, expertID, programID)
		if err != nil {
			return fmt.Errorf("target user is not an assigned expert for this program")
		}
	}

	// 4. Проверяем неизменяемость: если оценки уже COMPLETED, нельзя перезаписать
	existingEvals, err := uc.repo.ListEvaluations(ctx, applicantID)
	if err == nil {
		for _, e := range existingEvals {
			if e.ExpertID == expertID && e.Status == entity.EvaluationStatusCompleted && !isAdmin {
				return domain.ErrEvaluationImmutable
			}
		}
	}

	// 5. Валидация баллов по критериям
	criteria, err := uc.repo.GetCriteria(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch criteria: %w", err)
	}
	criteriaMap := make(map[string]entity.EvaluationCriteria)
	for _, c := range criteria {
		criteriaMap[c.Code] = c
	}

	status := entity.EvaluationStatusDraft
	if complete {
		status = entity.EvaluationStatusCompleted
	}

	sourceInfo := fmt.Sprintf("%s (%s)", role, userName)
	if isAISystem {
		sourceInfo = "AI Portfolio Scorer"
	}

	// 6. Формируем батч
	toSave := make([]entity.ExpertEvaluation, 0, len(evaluations))
	for _, e := range evaluations {
		c, ok := criteriaMap[e.Category]
		if !ok {
			return fmt.Errorf("unknown criteria: %s", e.Category)
		}
		if e.Score > c.MaxScore || e.Score < 0 {
			return domain.ErrScoreExceedsMax
		}

		toSave = append(toSave, entity.ExpertEvaluation{
			ApplicantID:     applicantID,
			ExpertID:        expertID,
			Category:        e.Category,
			Score:           e.Score,
			Comment:         e.Comment,
			Status:          status,
			UpdatedByID:     userID,
			IsAdminOverride: isAdmin,
			SourceInfo:      sourceInfo,
		})
	}

	// 7. Сохраняем батч
	if err := uc.repo.SaveEvaluationBatch(ctx, toSave); err != nil {
		return err
	}

	// 8. Если завершено — запускаем агрегацию (только для человеческих экспертов)
	if complete && !isAISystem {
		return uc.triggerAggregation(ctx, applicantID, programID)
	}

	return nil
}

func (uc *ExpertUseCase) ListExpertEvaluations(ctx context.Context, applicantID int64, currentUserID string, role string) ([]entity.ExpertEvaluation, error) {
	evals, err := uc.repo.ListEvaluations(ctx, applicantID)
	if err != nil {
		return nil, err
	}

	// Администратор видит все оценки, включая черновики — без ограничений.
	isAdmin := role == "admin"
	if isAdmin {
		return evals, nil
	}

	// Blind Grading: если эксперт ещё не завершил свою оценку — скрываем баллы других.
	// AI_SYSTEM оценки всегда видны как предзаполненные черновики.
	currentUserCompleted := false
	for _, e := range evals {
		if e.ExpertID == currentUserID && e.Status == entity.EvaluationStatusCompleted {
			currentUserCompleted = true
			break
		}
	}

	if !currentUserCompleted {
		for i := range evals {
			// Скрываем баллы других людей, но НЕ скрываем AI_SYSTEM
			if evals[i].ExpertID != currentUserID && evals[i].ExpertID != "AI_SYSTEM" {
				evals[i].Score = -1
				evals[i].Comment = "Score hidden until you complete your evaluation"
			}
		}
	}

	return evals, nil
}

func (uc *ExpertUseCase) AssignExpertSlot(ctx context.Context, userID string, slotNumber int, programID int64, requesterRole string) error {
	if requesterRole != "admin" && requesterRole != "manager" {
		return fmt.Errorf("only admins and managers can assign expert slots")
	}
	if programID <= 0 {
		return fmt.Errorf("program_id is required")
	}

	fmt.Printf("[DEBUG] AssignExpertSlot: userID=%s, slotNumber=%d, programID=%d, role=%s\n", userID, slotNumber, programID, requesterRole)

	if userID == "" || userID == "0" {
		return uc.repo.RemoveExpertSlot(ctx, slotNumber, programID)
	}

	// Проверяем лимит 3 слота per-программе
	slots, err := uc.repo.GetExpertSlots(ctx, programID)
	if err == nil && len(slots) >= 3 {
		exists := false
		for _, s := range slots {
			if s.SlotNumber == slotNumber {
				exists = true
				break
			}
		}
		if !exists {
			return fmt.Errorf("cannot have more than 3 active experts per program")
		}
	}

	return uc.repo.AssignExpertSlot(ctx, userID, slotNumber, programID)
}

func (uc *ExpertUseCase) GetExpertSlots(ctx context.Context, programID int64) ([]entity.ExpertSlot, error) {
	return uc.repo.GetExpertSlots(ctx, programID)
}

func (uc *ExpertUseCase) ListExperts(ctx context.Context) ([]entity.User, error) {
	return uc.repo.GetUsersByRoles(ctx, []string{"expert"})
}

// TriggerAIScoring реализует интерфейс AIScoringTrigger.
// Вызывается из applicant_usecase.go как горутина после TransferToExperts.
func (uc *ExpertUseCase) TriggerAIScoring(ctx context.Context, applicantID int64, programID int64) {
	// Используем отдельный контекст с таймаутом — запрос HTTP может занять до 10 минут
	scoringCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	fmt.Printf("[AI Scoring] Starting for applicant %d, program %d\n", applicantID, programID)

	// 1. Определяем схему оценивания по достижениям абитуриента
	scheme := uc.detectScheme(scoringCtx, applicantID)
	fmt.Printf("[AI Scoring] Detected scheme: %s\n", scheme)

	// 2. Загружаем критерии для этой схемы
	allCriteria, err := uc.repo.GetCriteria(scoringCtx)
	if err != nil {
		fmt.Printf("[AI Scoring] Failed to load criteria: %v\n", err)
		return
	}
	var schemeCriteria []entity.EvaluationCriteria
	for _, c := range allCriteria {
		if c.Scheme == scheme {
			schemeCriteria = append(schemeCriteria, c)
		}
	}
	if len(schemeCriteria) == 0 {
		fmt.Printf("[AI Scoring] No criteria found for scheme %s\n", scheme)
		return
	}

	// 3. Собираем данные абитуриента по всем категориям
	applicantData := uc.collectApplicantData(scoringCtx, applicantID)

	// 4. Вызываем Python-сервис скоринга
	if uc.scoringClient == nil {
		fmt.Printf("[AI Scoring] No scoring client configured, skipping\n")
		return
	}
	results, err := uc.scoringClient.ScorePortfolio(scoringCtx, schemeCriteria, applicantData)
	if err != nil {
		fmt.Printf("[AI Scoring] ScorePortfolio error for applicant %d: %v\n", applicantID, err)
		return
	}

	// 5. Формируем батч ExpertEvaluation с expert_id="AI_SYSTEM"
	batch := make([]entity.ExpertEvaluation, 0, len(results))
	for _, r := range results {
		batch = append(batch, entity.ExpertEvaluation{
			ApplicantID:     applicantID,
			ExpertID:        "AI_SYSTEM",
			Category:        r.Code,
			Score:           r.Score,
			Comment:         r.Comment,
			Status:          entity.EvaluationStatusDraft,
			UpdatedByID:     "AI_SYSTEM",
			IsAdminOverride: false,
			IsAIGenerated:   true,
			SourceInfo:      "AI Portfolio Scorer",
		})
	}

	if err := uc.repo.SaveEvaluationBatch(scoringCtx, batch); err != nil {
		fmt.Printf("[AI Scoring] Failed to save scores for applicant %d: %v\n", applicantID, err)
		return
	}

	fmt.Printf("[AI Scoring] Saved %d scores for applicant %d\n", len(batch), applicantID)
}

// GetEvaluationCriteriaForApplicant возвращает критерии, отфильтрованные по схеме абитуриента.
// Также возвращает название активной схемы ("default" или "ieee").
func (uc *ExpertUseCase) GetEvaluationCriteriaForApplicant(ctx context.Context, applicantID int64) ([]entity.EvaluationCriteria, string, error) {
	scheme := uc.resolveScheme(ctx, applicantID)

	allCriteria, err := uc.repo.GetCriteria(ctx)
	if err != nil {
		return nil, scheme, err
	}

	filtered := make([]entity.EvaluationCriteria, 0, len(allCriteria))
	for _, c := range allCriteria {
		if c.Scheme == scheme {
			filtered = append(filtered, c)
		}
	}
	return filtered, scheme, nil
}

// GetApplicantScoringScheme возвращает эффективную схему для абитуриента.
func (uc *ExpertUseCase) GetApplicantScoringScheme(ctx context.Context, applicantID int64) (string, error) {
	return uc.resolveScheme(ctx, applicantID), nil
}

// SetApplicantScoringScheme устанавливает схему оценивания вручную (только для admin).
func (uc *ExpertUseCase) SetApplicantScoringScheme(ctx context.Context, applicantID int64, scheme string, role string) error {
	if role != "admin" {
		return fmt.Errorf("only admins can override the scoring scheme")
	}
	if scheme != "auto" && scheme != "default" && scheme != "ieee" {
		return fmt.Errorf("invalid scheme '%s': must be one of auto, default, ieee", scheme)
	}
	return uc.appRepo.SetScoringScheme(ctx, applicantID, scheme)
}

// resolveScheme возвращает актуальную схему: берёт override из БД, иначе авто-детект.
func (uc *ExpertUseCase) resolveScheme(ctx context.Context, applicantID int64) string {
	stored, err := uc.appRepo.GetScoringScheme(ctx, applicantID)
	if err == nil && stored != "auto" && stored != "" {
		return stored
	}
	return uc.detectScheme(ctx, applicantID)
}

// detectScheme определяет схему оценивания: "ieee" или "default"
func (uc *ExpertUseCase) detectScheme(ctx context.Context, applicantID int64) string {
	achievements, err := uc.appRepo.ListAchievements(ctx, applicantID, "")
	if err != nil {
		return "default"
	}
	for _, a := range achievements {
		titleLower := strings.ToLower(a.AchievementTitle)
		descLower := strings.ToLower(a.Description)
		typeLower := strings.ToLower(a.AchievementType)
		if strings.Contains(titleLower, "ieee") || strings.Contains(descLower, "ieee") ||
			typeLower == "international_competition" || strings.Contains(titleLower, "international") {
			return "ieee"
		}
	}
	return "default"
}

// collectApplicantData собирает данные абитуриента по всем категориям в map для передачи в AI
func (uc *ExpertUseCase) collectApplicantData(ctx context.Context, applicantID int64) map[string]interface{} {
	data := make(map[string]interface{})

	if ident, err := uc.appRepo.GetLatestIdentification(ctx, applicantID); err == nil {
		data["identification"] = ident
	}
	if edu, err := uc.appRepo.GetLatestEducation(ctx, applicantID); err == nil {
		data["diploma"] = edu
	}
	if transcript, err := uc.appRepo.GetLatestTranscript(ctx, applicantID); err == nil {
		data["transcript"] = transcript
	}
	if lang, err := uc.appRepo.GetLatestLanguageTraining(ctx, applicantID); err == nil {
		data["language"] = lang
	}
	if motivation, err := uc.appRepo.GetLatestMotivation(ctx, applicantID); err == nil {
		data["motivation"] = motivation
	}
	if works, err := uc.appRepo.ListWorkExperience(ctx, applicantID, ""); err == nil {
		data["work"] = works
	}
	if recommendations, err := uc.appRepo.ListRecommendations(ctx, applicantID); err == nil {
		data["recommendation"] = recommendations
	}
	if achievements, err := uc.appRepo.ListAchievements(ctx, applicantID, ""); err == nil {
		data["achievement"] = achievements
	}
	if video, err := uc.appRepo.GetLatestVideo(ctx, applicantID); err == nil {
		data["video"] = video
	}
	if eduList, err := uc.appRepo.ListEducation(ctx, applicantID); err == nil {
		data["second_diploma"] = eduList
	}
	if profDev, err := uc.appRepo.ListWorkExperience(ctx, applicantID, "prof_development"); err == nil {
		data["prof_development"] = profDev
	}
	if certs, err := uc.appRepo.ListAchievements(ctx, applicantID, "certification"); err == nil {
		data["certification"] = certs
	}

	// Сериализуем в map[string]interface{} через JSON для единого формата
	dataJSON, _ := json.Marshal(data)
	var result map[string]interface{}
	_ = json.Unmarshal(dataJSON, &result)
	return result
}

func (uc *ExpertUseCase) triggerAggregation(ctx context.Context, applicantID int64, programID int64) error {
	// 1. Проверяем, все ли назначенные эксперты (только люди) завершили оценку
	slots, err := uc.repo.GetExpertSlots(ctx, programID)
	if err != nil {
		return err
	}

	evals, err := uc.repo.ListEvaluations(ctx, applicantID)
	if err != nil {
		return err
	}

	// Считаем только завершённые оценки людей (не AI_SYSTEM)
	completedExperts := make(map[string]bool)
	for _, e := range evals {
		if e.Status == entity.EvaluationStatusCompleted && e.ExpertID != "AI_SYSTEM" {
			completedExperts[e.ExpertID] = true
		}
	}

	assignedExpertsCount := 0
	for _, s := range slots {
		if s.UserID != "" {
			assignedExpertsCount++
		}
	}

	// Все назначенные эксперты должны завершить оценку
	allDone := true
	for _, s := range slots {
		if s.UserID != "" && !completedExperts[s.UserID] {
			allDone = false
			break
		}
	}

	if !(allDone && assignedExpertsCount > 0) {
		return uc.appRepo.UpdateApplicantRanking(ctx, applicantID, 0, entity.ApplicantStatusEvaluation)
	}

	// 2. Определяем схему и загружаем коды критериев для агрегации
	scheme := uc.resolveScheme(ctx, applicantID)
	allCriteria, err := uc.repo.GetCriteria(ctx)
	if err != nil {
		return fmt.Errorf("triggerAggregation: failed to load criteria: %w", err)
	}

	categoryCodes := make([]string, 0, len(allCriteria))
	for _, c := range allCriteria {
		if c.Scheme == scheme {
			categoryCodes = append(categoryCodes, c.Code)
		}
	}

	if len(categoryCodes) == 0 {
		return fmt.Errorf("triggerAggregation: no criteria found for scheme %s", scheme)
	}

	// 3. Считаем агрегированный балл только по категориям активной схемы
	score, err := uc.repo.GetAggregatedScore(ctx, applicantID, categoryCodes)
	if err != nil {
		return fmt.Errorf("triggerAggregation: GetAggregatedScore: %w", err)
	}

	return uc.appRepo.UpdateApplicantRanking(ctx, applicantID, score, entity.ApplicantStatusEvaluated)
}
