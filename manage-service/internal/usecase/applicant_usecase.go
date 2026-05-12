package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"manage-service/internal/domain/entity"
)

type ApplicantUseCase struct {
	repo       ApplicantRepo
	docRepo    DocumentRepo
	docUC      Document
	expertRepo ExpertRepo
	aiScoring  AIScoringTrigger // горутина AI-оценивания после TransferToExperts
}

func NewApplicantUseCase(repo ApplicantRepo, docRepo DocumentRepo, docUC Document, expertRepo ExpertRepo) *ApplicantUseCase {
	return &ApplicantUseCase{repo: repo, docRepo: docRepo, docUC: docUC, expertRepo: expertRepo}
}

// SetAIScoringTrigger подключает AI-оценивание. Вызывается в main.go после создания ExpertUseCase.
func (uc *ApplicantUseCase) SetAIScoringTrigger(trigger AIScoringTrigger) {
	uc.aiScoring = trigger
}

func (uc *ApplicantUseCase) CreateApplicant(ctx context.Context, programID int64, firstName, lastName, patronymic string) (entity.Applicant, error) {
	applicant := entity.Applicant{
		ProgramID: programID,
		Status:    "uploaded",
	}

	err := uc.repo.Store(ctx, &applicant)
	if err != nil {
		return entity.Applicant{}, fmt.Errorf("UseCase - CreateApplicant - uc.repo.Store: %w", err)
	}

	// Create initial identification record (operator source V1)
	ident := entity.IdentificationData{
		ApplicantID: applicant.ID,
		Name:        firstName,
		Surname:     lastName,
		Patronymic:  patronymic,
		Source:      "operator",
	}
	err = uc.repo.StoreIdentification(ctx, ident)
	if err != nil {
		fmt.Printf("Error storing identification for applicant %d: %v\n", applicant.ID, err)
		// We can decide whether to fail the whole creation or not.
		// For now, let's return error to be sure.
		return applicant, fmt.Errorf("UseCase - CreateApplicant - uc.repo.StoreIdentification: %w", err)
	}

	return applicant, nil
}

func (uc *ApplicantUseCase) GetApplicantData(ctx context.Context, applicantID int64, category string) (interface{}, error) {
	switch category {
	case "passport", "personal_data", "resume":
		return uc.repo.GetLatestIdentification(ctx, applicantID)
	case "diploma":
		return uc.repo.GetLatestEducation(ctx, applicantID)
	case "work", "prof_development":
		return uc.repo.ListWorkExperience(ctx, applicantID, category)
	case "recommendation":
		return uc.repo.ListRecommendations(ctx, applicantID)
	case "achievement", "certification":
		return uc.repo.ListAchievements(ctx, applicantID, category)
	case "transcript":
		return uc.repo.GetLatestTranscript(ctx, applicantID)
	case "language":
		return uc.repo.GetLatestLanguageTraining(ctx, applicantID)
	case "motivation":
		return uc.repo.GetLatestMotivation(ctx, applicantID)
	case "video_presentation":
		return uc.repo.GetLatestVideo(ctx, applicantID)
	case "second_diploma":
		return uc.repo.ListEducation(ctx, applicantID)
	default:
		return nil, fmt.Errorf("unknown category: %s", category)
	}
}

func (uc *ApplicantUseCase) ListApplicants(ctx context.Context, programID int64) ([]entity.Applicant, error) {
	applicants, err := uc.repo.List(ctx, programID)
	if err != nil {
		return nil, fmt.Errorf("UseCase - ListApplicants - uc.repo.List: %w", err)
	}
	return applicants, nil
}

func (uc *ApplicantUseCase) DeleteApplicant(ctx context.Context, id int64) error {
	// 1. Fetch all documents for this applicant to delete from MinIO
	docs, err := uc.docRepo.GetDocuments(ctx, id)
	if err == nil {
		for _, doc := range docs {
			_ = uc.docUC.DeleteDocument(ctx, id, doc.ID)
		}
	}

	// 2. Delete the applicant from DB
	return uc.repo.Delete(ctx, id)
}

func (uc *ApplicantUseCase) DeleteApplicantData(ctx context.Context, applicantID int64, category string, dataID int64) error {
	switch category {
	case "work", "prof_development":
		return uc.repo.DeleteWorkExperience(ctx, dataID)
	case "recommendation":
		return uc.repo.DeleteRecommendation(ctx, dataID)
	case "achievement", "certification":
		return uc.repo.DeleteAchievement(ctx, dataID)
	case "language":
		return uc.repo.DeleteLanguageTraining(ctx, dataID)
	default:
		return fmt.Errorf("UseCase - DeleteApplicantData - unsupported category: %s", category)
	}
}

func (uc *ApplicantUseCase) UpdateApplicantData(ctx context.Context, applicantID int64, category string, rawData map[string]interface{}) error {
	// 1. Construct Source string: роль (ФИО)
	var operatorInfo string
	role := getString(rawData, "role")
	if role == "" {
		role = "оператор"
	}
	fName := getString(rawData, "first_name")
	lName := getString(rawData, "last_name")
	pName := getString(rawData, "user_patronymic")

	fullName := strings.TrimSpace(fmt.Sprintf("%s %s %s", lName, fName, pName))
	if fullName != "" {
		operatorInfo = fmt.Sprintf("%s (%s)", role, fullName)
	} else {
		// Fallback to what was there before if no name provided
		if opName := getString(rawData, "operator_name"); opName != "" && opName != "operator" {
			operatorInfo = fmt.Sprintf("%s (%s)", role, opName)
		} else {
			operatorInfo = role
		}
	}

	parseTime := func(val interface{}) time.Time {
		if s, ok := val.(string); ok && s != "" {
			// Try common formats
			if t, err := time.Parse("2006-01-02", s[:10]); err == nil {
				return t
			}
			// Russian format from UI: DD.MM.YYYY
			if len(s) >= 10 {
				parts := strings.Split(s[:10], ".")
				if len(parts) == 3 {
					iso := fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
					if t, err := time.Parse("2006-01-02", iso); err == nil {
						return t
					}
				}
			}
		}
		return time.Time{}
	}

	switch category {
	case "passport", "personal_data", "resume":
		latest, err := uc.repo.GetLatestIdentification(ctx, applicantID)
		if err != nil {
			return err
		}
		latest.ApplicantID = applicantID
		latest.Name = getStringWithFallback(rawData, "name", latest.Name)
		latest.Surname = getStringWithFallback(rawData, "surname", latest.Surname)
		latest.Patronymic = getStringWithFallback(rawData, "patronymic", latest.Patronymic)
		latest.Email = getStringWithFallback(rawData, "email", latest.Email)
		latest.Phone = getStringWithFallback(rawData, "phone", latest.Phone)
		latest.DocumentNumber = getStringWithFallback(rawData, "document_number", latest.DocumentNumber)
		latest.Gender = getStringWithFallback(rawData, "gender", latest.Gender)
		latest.Nationality = getStringWithFallback(rawData, "nationality", latest.Nationality)
		latest.Source = operatorInfo

		if dobRaw, ok := rawData["date_of_birth"]; ok {
			latest.DateOfBirth = parseTime(dobRaw)
		}

		return uc.repo.UpdateIdentification(ctx, latest)

	case "diploma":
		latest, err := uc.repo.GetLatestEducation(ctx, applicantID)
		if err != nil {
			return err
		}
		latest.ApplicantID = applicantID
		latest.InstitutionName = getStringWithFallback(rawData, "institution_name", latest.InstitutionName)
		latest.DegreeTitle = getStringWithFallback(rawData, "degree_title", latest.DegreeTitle)
		latest.Major = getStringWithFallback(rawData, "major", latest.Major)
		latest.DiplomaSerialNumber = getStringWithFallback(rawData, "diploma_serial_number", latest.DiplomaSerialNumber)
		latest.Source = operatorInfo

		if gradRaw, ok := rawData["graduation_date"]; ok {
			latest.GraduationDate = parseTime(gradRaw)
		}

		return uc.repo.UpdateEducation(ctx, latest)

	case "transcript":
		latest, err := uc.repo.GetLatestTranscript(ctx, applicantID)
		if err != nil {
			return err
		}
		latest.ApplicantID = applicantID
		latest.CumulativeGPA = getFloat(rawData, "cumulative_gpa", latest.CumulativeGPA)
		latest.CumulativeGrade = getStringWithFallback(rawData, "cumulative_grade", latest.CumulativeGrade)
		latest.TotalCredits = getFloat(rawData, "total_credits", latest.TotalCredits)
		latest.ObtainedCredits = getFloat(rawData, "obtained_credits", latest.ObtainedCredits)
		latest.TotalSemesters = getInt(rawData, "total_semesters", latest.TotalSemesters)
		latest.Source = operatorInfo

		return uc.repo.UpdateTranscript(ctx, latest)

	case "language":
		latest, err := uc.repo.GetLatestLanguageTraining(ctx, applicantID)
		if err != nil {
			return err
		}
		latest.ApplicantID = applicantID
		latest.RussianLevel = getStringWithFallback(rawData, "russian_level", latest.RussianLevel)
		latest.EnglishLevel = getStringWithFallback(rawData, "english_level", latest.EnglishLevel)
		latest.ExamName = getStringWithFallback(rawData, "exam_name", latest.ExamName)
		latest.Score = getStringWithFallback(rawData, "score", latest.Score)
		latest.Source = operatorInfo

		return uc.repo.UpdateLanguageTraining(ctx, latest)

	case "motivation":
		latest, err := uc.repo.GetLatestMotivation(ctx, applicantID)
		if err != nil {
			return err
		}
		latest.ApplicantID = applicantID
		latest.ReasonsForApplying = getStringWithFallback(rawData, "reasons_for_applying", latest.ReasonsForApplying)
		latest.ExperienceSummary = getStringWithFallback(rawData, "experience_summary", latest.ExperienceSummary)
		latest.CareerGoals = getStringWithFallback(rawData, "career_goals", latest.CareerGoals)
		latest.DetectedLanguage = getStringWithFallback(rawData, "detected_language", latest.DetectedLanguage)
		latest.MainText = getStringWithFallback(rawData, "main_text", latest.MainText)
		latest.Source = operatorInfo

		return uc.repo.UpdateMotivation(ctx, latest)

	case "video_presentation":
		latest, err := uc.repo.GetLatestVideo(ctx, applicantID)
		if err != nil {
			return err
		}
		latest.ApplicantID = applicantID
		latest.VideoURL = getStringWithFallback(rawData, "video_url", latest.VideoURL)
		latest.Source = operatorInfo

		return uc.repo.UpdateVideo(ctx, latest)

	case "work", "prof_development", "recommendation", "achievement", "certification", "second_diploma":
		recordsRaw, ok := rawData["records"].([]interface{})
		if !ok {
			return fmt.Errorf("UseCase - UpdateApplicantData - records field is missing or invalid for category %s", category)
		}

		for _, recRaw := range recordsRaw {
			m, ok := recRaw.(map[string]interface{})
			if !ok {
				continue
			}

			id := int64(getFloat(m, "id", 0))
			if id == 0 {
				continue
			}
			switch category {
			case "work", "prof_development":
				startDate := parseTime(m["start_date"])
				endDate := parseTime(m["end_date"])
				var endDatePtr *time.Time
				if !endDate.IsZero() {
					endDatePtr = &endDate
				}

				wExp := entity.WorkExperience{
					ID:          id,
					CompanyName: getString(m, "company_name"),
					Position:    getString(m, "position"),
					StartDate:   startDate,
					EndDate:     endDatePtr,
					Country:     getString(m, "country"),
					City:        getString(m, "city"),
					RecordType:  getString(m, "record_type"),
					Source:      operatorInfo,
				}
				_ = uc.repo.UpdateWorkExperience(ctx, wExp)

			case "recommendation":
				rec := entity.RecommendationData{
					ID:                id,
					AuthorName:        getString(m, "author_name"),
					AuthorPosition:    getString(m, "author_position"),
					AuthorInstitution: getString(m, "author_institution"),
					KeyStrengths:      getString(m, "key_strengths"),
					Source:            operatorInfo,
				}
				_ = uc.repo.UpdateRecommendation(ctx, rec)

			case "achievement", "certification":
				date := parseTime(m["date_received"])
				ach := entity.AchievementData{
					ID:               id,
					AchievementTitle: getString(m, "achievement_title"),
					Description:      getString(m, "description"),
					DateReceived:     date,
					AchievementType:  getString(m, "achievement_type"),
					Source:           operatorInfo,
				}
				_ = uc.repo.UpdateAchievement(ctx, ach)

			case "second_diploma":
				gradDate := parseTime(m["graduation_date"])
				edu := entity.EducationData{
					ID:                  id,
					InstitutionName:     getString(m, "institution_name"),
					DegreeTitle:         getString(m, "degree_title"),
					Major:               getString(m, "major"),
					GraduationDate:      gradDate,
					DiplomaSerialNumber: getString(m, "diploma_serial_number"),
					Source:              operatorInfo,
				}
				_ = uc.repo.UpdateEducation(ctx, edu)
			}
		}
		return nil
	}
	return fmt.Errorf("UseCase - UpdateApplicantData - unknown category: %s", category)
}

func (uc *ApplicantUseCase) TransferToExperts(ctx context.Context, applicantID int64, confirmedBy string) error {
	if confirmedBy != "" {
		if err := uc.repo.ConfirmModelData(ctx, applicantID, confirmedBy); err != nil {
			return err
		}
	}
	if err := uc.repo.UpdateApplicantRanking(ctx, applicantID, 0, entity.ApplicantStatusEvaluation); err != nil {
		return err
	}

	// Запускаем AI-оценивание в горутине после смены статуса.
	// Используем context.Background() — контекст запроса отменится после ответа HTTP.
	if uc.aiScoring != nil {
		applicant, err := uc.repo.GetByID(ctx, applicantID)
		if err == nil {
			go uc.aiScoring.TriggerAIScoring(context.Background(), applicantID, applicant.ProgramID)
		}
	}
	return nil
}

func (uc *ApplicantUseCase) TransferToOperator(ctx context.Context, applicantID int64) error {
	// 1. Собираем загруженные документы
	docs, err := uc.docRepo.GetDocuments(ctx, applicantID)
	if err != nil {
		return err
	}
	uploadedTypes := make(map[string]bool)
	for _, d := range docs {
		if d.Status == "completed" || d.Status == "processing" {
			uploadedTypes[d.FileType] = true
		}
	}
	// Видео может быть ссылкой, а не файлом
	video, err := uc.repo.GetLatestVideo(ctx, applicantID)
	if err == nil && video.VideoURL != "" {
		uploadedTypes["video_presentation"] = true
	}

	// 2. Определяем схему абитуриента (default / ieee)
	scheme := "default"
	achievements, _ := uc.repo.ListAchievements(ctx, applicantID, "")
	for _, a := range achievements {
		if strings.EqualFold(a.AchievementTitle, "ieee") || strings.EqualFold(a.Description, "ieee") {
			scheme = "ieee"
			break
		}
	}

	// 3. Читаем обязательные критерии из БД
	allCriteria, err := uc.expertRepo.GetCriteria(ctx)
	if err != nil {
		return fmt.Errorf("TransferToOperator: failed to load criteria: %w", err)
	}

	// 4. Проверяем наличие обязательных документов
	var missing []string
	for _, c := range allCriteria {
		if !c.IsMandatory || c.Scheme != scheme {
			continue
		}
		// Хотя бы один тип документа из списка должен быть загружен
		found := false
		for _, dt := range c.DocumentTypes {
			if uploadedTypes[dt] {
				found = true
				break
			}
		}
		if !found && len(c.DocumentTypes) > 0 {
			missing = append(missing, c.Title)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing mandatory documents: %v", missing)
	}

	return uc.repo.UpdateApplicantRanking(ctx, applicantID, 0, entity.ApplicantStatusVerification)
}
