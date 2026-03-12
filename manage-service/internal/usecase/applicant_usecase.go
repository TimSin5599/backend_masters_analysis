package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"manage-service/internal/entity"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

type ApplicantUseCase struct {
	repo      ApplicantRepo
	queue     DocumentQueueRepo
	producer  DocumentQueueProducer
	extractor ExtractionClient
	s3        S3Provider // Assuming we'll have an interface for MinIO
}

func New(r ApplicantRepo, q DocumentQueueRepo, p DocumentQueueProducer, e ExtractionClient, s3 S3Provider) *ApplicantUseCase {
	return &ApplicantUseCase{
		repo:      r,
		queue:     q,
		producer:  p,
		extractor: e,
		s3:        s3,
	}
}

// CreateApplicant - Регистрация анкеты абитуриента менеджером
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
		Version:     1,
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

// GetApplicantData
func (uc *ApplicantUseCase) GetApplicantData(ctx context.Context, id int64, category string) (interface{}, error) {
	switch category {
	case "passport":
		return uc.repo.GetLatestIdentification(ctx, id)
	case "diploma":
		return uc.repo.GetLatestEducation(ctx, id)
	case "work":
		return uc.repo.ListWorkExperience(ctx, id)
	case "recommendation":
		return uc.repo.ListRecommendations(ctx, id)
	case "achievement":
		return uc.repo.ListAchievements(ctx, id)
	case "transcript":
		return uc.repo.GetLatestTranscript(ctx, id)
	case "language":
		return uc.repo.GetLatestLanguageTraining(ctx, id)
	case "motivation":
		return uc.repo.GetLatestMotivation(ctx, id)
	default:
		return nil, fmt.Errorf("unknown category: %s", category)
	}
}

// ListApplicants - Получение списка всех абитуриентов (с опциональным фильтром по программе)
func (uc *ApplicantUseCase) ListApplicants(ctx context.Context, programID int64) ([]entity.Applicant, error) {
	applicants, err := uc.repo.List(ctx, programID)
	if err != nil {
		return nil, fmt.Errorf("UseCase - ListApplicants - uc.repo.List: %w", err)
	}
	return applicants, nil
}

func (uc *ApplicantUseCase) ListPrograms(ctx context.Context) ([]entity.Program, error) {
	return uc.repo.ListPrograms(ctx)
}

func (uc *ApplicantUseCase) DeleteApplicant(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *ApplicantUseCase) GetProgram(ctx context.Context, id int64) (entity.Program, error) {
	return uc.repo.GetProgramByID(ctx, id)
}

// convertImageToPDF converts an image to a single-page PDF.
func convertImageToPDF(fileName string, content []byte) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	ext := strings.ToLower(filepath.Ext(fileName))
	imageType := "JPG"
	if ext == ".png" {
		imageType = "PNG"
	}

	opt := gofpdf.ImageOptions{ImageType: imageType, ReadDpi: true, AllowNegativePosition: true}
	info := pdf.RegisterImageOptionsReader(fileName, opt, bytes.NewReader(content))
	if info == nil {
		return nil, fmt.Errorf("failed to register image with gofpdf")
	}

	pageW, pageH := pdf.GetPageSize()
	maxWidth := pageW - 20
	maxHeight := pageH - 20

	imgW := info.Width()
	imgH := info.Height()

	scale := maxWidth / imgW
	if (maxHeight / imgH) < scale {
		scale = maxHeight / imgH
	}

	finalW := imgW * scale
	finalH := imgH * scale

	x := (pageW - finalW) / 2
	y := (pageH - finalH) / 2

	pdf.ImageOptions(fileName, x, y, finalW, finalH, false, opt, 0, "")

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UploadDocument - Загрузка документа и триггер ИИ
func (uc *ApplicantUseCase) UploadDocument(ctx context.Context, applicantID int64, category string, fileName string, content []byte) error {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
		pdfBytes, err := convertImageToPDF(fileName, content)
		if err != nil {
			return fmt.Errorf("failed to convert image to pdf: %w", err)
		}
		content = pdfBytes
		fileName = fileName[:len(fileName)-len(ext)] + ".pdf"
	}

	// 1. Сохранение в MinIO
	storagePath := fmt.Sprintf("applicants/%d/%s/%s", applicantID, category, fileName)
	err := uc.s3.UploadFile(ctx, storagePath, content)
	if err != nil {
		return fmt.Errorf("UseCase - UploadDocument - s3.UploadFile: %w", err)
	}

	// 2. Регистрация в БД
	doc := entity.Document{
		ApplicantID: applicantID,
		FileType:    category, // Map category to FileType for now
		StoragePath: storagePath,
		FileName:    fileName,
		Status:      "processing",
	}

	err = uc.repo.StoreDocument(ctx, &doc)
	if err != nil {
		return fmt.Errorf("UseCase - UploadDocument - uc.repo.StoreDocument: %w", err)
	}

	// 3. Добавление задачи в очередь (RabbitMQ & PostgreSQL)
	priority := 5
	switch category {
	case "passport":
		priority = 10
	case "diploma", "transcript":
		priority = 9
	case "language":
		priority = 8
	case "experience":
		priority = 7
	case "recommendation":
		priority = 6
	case "achievement":
		priority = 5
	case "motivation":
		priority = 4
	}

	task := entity.DocumentQueueTask{
		ApplicantID:      applicantID,
		DocumentCategory: category,
		FilePath:         storagePath,
		Priority:         priority,
		Status:           "pending",
	}

	queueID, err := uc.queue.Enqueue(ctx, task)
	if err != nil {
		return fmt.Errorf("UseCase - UploadDocument - queue.Enqueue: %w", err)
	}

	task.ID = queueID
	err = uc.producer.PublishTask(task)
	if err != nil {
		// If publishing fails, mark as failed in DB
		errMsg := err.Error()
		_ = uc.queue.UpdateStatus(context.Background(), queueID, "failed", &errMsg)
		return fmt.Errorf("UseCase - UploadDocument - producer.PublishTask: %w", err)
	}

	return nil
}

func (uc *ApplicantUseCase) ReprocessDocument(ctx context.Context, documentID int64) error {
	// 1. Get document metadata
	doc, err := uc.repo.GetDocumentByID(ctx, documentID)
	if err != nil {
		return fmt.Errorf("UseCase - ReprocessDocument - repo.GetDocumentByID: %w", err)
	}

	// 2. Reset status to processing
	err = uc.repo.UpdateDocumentStatus(ctx, doc.ID, "processing")
	if err != nil {
		return fmt.Errorf("UseCase - ReprocessDocument - repo.UpdateDocumentStatus: %w", err)
	}

	// 3. Добавление задачи в очередь (RabbitMQ & PostgreSQL)
	priority := 10 // Повторное сканирование получает высокий приоритет

	task := entity.DocumentQueueTask{
		ApplicantID:      doc.ApplicantID,
		DocumentCategory: doc.FileType, // We mapped category to FileType earlier
		FilePath:         doc.StoragePath,
		Priority:         priority,
		Status:           "pending",
	}

	queueID, err := uc.queue.Enqueue(ctx, task)
	if err != nil {
		return fmt.Errorf("UseCase - ReprocessDocument - queue.Enqueue: %w", err)
	}

	task.ID = queueID
	err = uc.producer.PublishTask(task)
	if err != nil {
		// If publishing fails, mark as failed in DB
		errMsg := err.Error()
		_ = uc.queue.UpdateStatus(context.Background(), queueID, "failed", &errMsg)
		return fmt.Errorf("UseCase - ReprocessDocument - producer.PublishTask: %w", err)
	}

	fmt.Printf("[USECASE] ✅ Reprocess task queued for doc %d (Applicant: %d)\n", doc.ID, doc.ApplicantID)

	return nil
}

func (uc *ApplicantUseCase) ReprocessLatestDocument(ctx context.Context, applicantID int64, category string) (int64, error) {
	// 1. Get latest document
	doc, err := uc.repo.GetLatestDocumentByCategory(ctx, applicantID, category)
	if err != nil {
		return 0, fmt.Errorf("UseCase - ReprocessLatestDocument - repo.GetLatestDocumentByCategory: %w", err)
	}

	// 2. Delegate to generic reprocess
	return doc.ID, uc.ReprocessDocument(ctx, doc.ID)
}

// ProcessAIResult - Обработка "сырых" данных из ИИ и перевод в типизированные таблицы (V1)
func (uc *ApplicantUseCase) ProcessAIResult(ctx context.Context, applicantID int64, documentID int64, category string, rawData map[string]string) error {
	// 0. Check for AI extraction errors
	if errStr, ok := rawData["error"]; ok {
		return fmt.Errorf("AI extraction failed: %s", errStr)
	}

	// 1. Сохраняем в лог extracted_fields
	for k, v := range rawData {
		_ = uc.repo.StoreExtractedField(ctx, applicantID, documentID, k, v)
	}

	// 2. В зависимости от категории наполняем конкретную таблицу (Version 1, source='model')

	// Helper functions scoped to the switch
	parseTimeSafe := func(s string) time.Time {
		if s == "" || len(s) < 10 {
			return time.Time{}
		}
		// Some models return date strings inconsistently.
		// Try a few formats or fallback
		t, e := time.Parse("2006-01-02", s[:10])
		if e != nil {
			return time.Time{} // Return zero time if it fails completely
		}
		return t
	}

	// 2. Clear old data for this exact document_id (UPSERT strategy via Delete-Then-Insert)
	_ = uc.repo.DeleteDataByDocumentID(ctx, category, documentID)

	parseFloatSafe := func(s string) float64 {
		if s == "" {
			return 0
		}
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	parseIntSafe := func(s string) int {
		if s == "" {
			return 0
		}
		i, _ := strconv.Atoi(s)
		return i
	}

	switch category {
	case "passport":
		dob := parseTimeSafe(rawData["date_of_birth"])
		docIDRef := &documentID
		data := entity.IdentificationData{
			ApplicantID:    applicantID,
			DocumentID:     docIDRef,
			Name:           rawData["name"],
			Surname:        rawData["surname"],
			Patronymic:     rawData["patronymic"],
			DocumentNumber: rawData["document_number"],
			Nationality:    rawData["nationality"],
			DateOfBirth:    dob,
			Source:         "model",
			Version:        1,
		}
		if err := uc.repo.StoreIdentification(ctx, data); err != nil {
			return err
		}

	case "diploma":
		gradDate := parseTimeSafe(rawData["graduation_date"])
		docIDRef := &documentID
		data := entity.EducationData{
			ApplicantID:         applicantID,
			DocumentID:          docIDRef,
			InstitutionName:     rawData["institution_name"],
			DegreeTitle:         rawData["degree_title"],
			Major:               rawData["major"],
			GraduationDate:      gradDate,
			DiplomaSerialNumber: rawData["diploma_serial_number"],
			Source:              "model",
			Version:             1,
		}
		if err := uc.repo.StoreEducation(ctx, data); err != nil {
			return err
		}

	case "recommendation":
		docIDRef := &documentID
		data := entity.RecommendationData{
			ApplicantID:       applicantID,
			DocumentID:        docIDRef,
			AuthorName:        rawData["author_name"],
			AuthorPosition:    rawData["author_position"],
			AuthorInstitution: rawData["author_institution"],
			KeyStrengths:      rawData["key_strengths"],
			Source:            "model",
			Version:           1,
		}
		if err := uc.repo.StoreRecommendation(ctx, data); err != nil {
			return err
		}

	case "achievement":
		docIDRef := &documentID
		dateReceived := parseTimeSafe(rawData["date_received"])
		data := entity.AchievementData{
			ApplicantID:      applicantID,
			DocumentID:       docIDRef,
			AchievementTitle: rawData["achievement_title"],
			Description:      rawData["description"],
			DateReceived:     dateReceived,
			AchievementType:  rawData["achievement_type"],
			Company:          rawData["company"],
			Source:           "model",
			Version:          1,
		}
		if err := uc.repo.StoreAchievement(ctx, data); err != nil {
			return err
		}

	case "language":
		docIDRef := &documentID
		data := entity.LanguageTraining{
			ApplicantID:  applicantID,
			DocumentID:   docIDRef,
			RussianLevel: rawData["russian_level"],
			EnglishLevel: rawData["english_level"],
			ExamName:     rawData["exam_name"],
			Score:        rawData["score"],
			Source:       "model",
			Version:      1,
		}
		if err := uc.repo.StoreLanguageTraining(ctx, data); err != nil {
			return err
		}

	case "motivation":
		docIDRef := &documentID
		data := entity.MotivationData{
			ApplicantID:        applicantID,
			DocumentID:         docIDRef,
			ReasonsForApplying: rawData["reasons_for_applying"],
			ExperienceSummary:  rawData["experience_summary"],
			CareerGoals:        rawData["career_goals"],
			DetectedLanguage:   rawData["detected_language"],
			MainText:           rawData["main_text"],
			Source:             "model",
			Version:            1,
		}
		if err := uc.repo.StoreMotivation(ctx, data); err != nil {
			return err
		}

	case "work":
		docIDRef := &documentID
		var experiences []map[string]interface{}
		if expRaw, ok := rawData["experiences"]; ok && expRaw != "" {
			_ = json.Unmarshal([]byte(expRaw), &experiences)
			for _, exp := range experiences {
				companyName, _ := exp["company_name"].(string)
				position, _ := exp["position"].(string)
				startDateStr, _ := exp["start_date"].(string)
				endDateStr, _ := exp["end_date"].(string)
				
				startDate := parseTimeSafe(startDateStr)
				endDate := parseTimeSafe(endDateStr)
				var endDatePtr *time.Time
				if !endDate.IsZero() {
					endDatePtr = &endDate
				}

				wExp := entity.WorkExperience{
					ApplicantID: applicantID,
					DocumentID:  docIDRef,
					CompanyName: companyName,
					Position:    position,
					StartDate:   startDate,
					EndDate:     endDatePtr,
					Country:     "",
					City:        "",
					Source:      "model",
					Version:     1,
				}
				if err := uc.repo.StoreWorkExperience(ctx, wExp); err != nil {
					return err
				}
			}
		}

	case "transcript":
		cumulativeGPA := parseFloatSafe(rawData["cumulative_gpa"])
		cumulativeGrade := rawData["cumulative_grade"]
		totalCredits := parseFloatSafe(rawData["total_credits"])
		obtainedCredits := parseFloatSafe(rawData["obtained_credits"])
		totalSemesters := parseIntSafe(rawData["total_semesters"])
		
		docIDRef := &documentID
		
		existing, err := uc.repo.GetTranscriptByDocumentID(ctx, documentID)
		if err == nil {
			// UPDATE existing
			existing.CumulativeGPA = cumulativeGPA
			existing.CumulativeGrade = cumulativeGrade
			existing.TotalCredits = totalCredits
			existing.ObtainedCredits = obtainedCredits
			existing.TotalSemesters = totalSemesters
			existing.Source = "model"
			err = uc.repo.UpdateTranscript(ctx, existing)
		} else {
			// INSERT new
			data := entity.TranscriptData{
				ApplicantID:     applicantID,
				DocumentID:      docIDRef,
				CumulativeGPA:   cumulativeGPA,
				CumulativeGrade: cumulativeGrade,
				TotalCredits:    totalCredits,
				ObtainedCredits: obtainedCredits,
				TotalSemesters:  totalSemesters,
				Source:          "model",
				Version:         1,
			}
			err = uc.repo.StoreTranscript(ctx, data)
		}
		if err != nil {
			return err
		}
	}

	// If we reached here, data was stored successfully. Update document status.
	return uc.repo.UpdateDocumentStatus(ctx, documentID, "completed")
}

func (uc *ApplicantUseCase) DeleteApplicantData(ctx context.Context, applicantID int64, category string, dataID int64) error {
	switch category {
	case "work":
		return uc.repo.DeleteWorkExperience(ctx, dataID)
	case "recommendation":
		return uc.repo.DeleteRecommendation(ctx, dataID)
	case "achievement":
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
	pName := getString(rawData, "patronymic")
	
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
	case "passport":
		latest, err := uc.repo.GetLatestIdentification(ctx, applicantID)
		if err != nil {
			return err
		}
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
		latest.ReasonsForApplying = getStringWithFallback(rawData, "reasons_for_applying", latest.ReasonsForApplying)
		latest.ExperienceSummary = getStringWithFallback(rawData, "experience_summary", latest.ExperienceSummary)
		latest.CareerGoals = getStringWithFallback(rawData, "career_goals", latest.CareerGoals)
		latest.DetectedLanguage = getStringWithFallback(rawData, "detected_language", latest.DetectedLanguage)
		latest.MainText = getStringWithFallback(rawData, "main_text", latest.MainText)
		latest.Source = operatorInfo
		
		return uc.repo.UpdateMotivation(ctx, latest)

	case "work", "recommendation", "achievement":
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
			case "work":
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

			case "achievement":
				date := parseTime(m["date_received"])
				ach := entity.AchievementData{
					ID:               id,
					AchievementTitle: getString(m, "achievement_title"),
					Description:      getString(m, "description"),
					DateReceived:     date,
					AchievementType:  getString(m, "achievement_type"),
					Company:          getString(m, "company"),
					Source:           operatorInfo,
				}
				_ = uc.repo.UpdateAchievement(ctx, ach)
			}
		}
		return nil
	}
	return fmt.Errorf("UseCase - UpdateApplicantData - unknown category: %s", category)
}

func (uc *ApplicantUseCase) ViewDocument(ctx context.Context, applicantID int64, category string) ([]byte, string, string, error) {
	// 1. Get latest document metadata for category
	doc, err := uc.repo.GetLatestDocumentByCategory(ctx, applicantID, category)
	if err != nil {
		return nil, "", "", fmt.Errorf("UseCase - ViewDocument - repo.GetLatestDocumentByCategory: %w", err)
	}

	// 2. Get file content from S3
	content, err := uc.s3.GetFile(ctx, doc.StoragePath)
	if err != nil {
		return nil, "", "", fmt.Errorf("UseCase - ViewDocument - s3.GetFile: %w", err)
	}

	// Determine generic content type based on category/file_type or extension
	contentType := "application/pdf"
	if doc.FileType == "passport" || doc.FileType == "diploma" {
		contentType = "application/pdf"
	}
	// Naive extension check
	if len(doc.FileName) > 4 && doc.FileName[len(doc.FileName)-4:] == ".jpg" {
		contentType = "image/jpeg"
	} else if len(doc.FileName) > 4 && doc.FileName[len(doc.FileName)-4:] == ".png" {
		contentType = "image/png"
	}

	return content, contentType, doc.FileName, nil
}

func (uc *ApplicantUseCase) ViewDocumentByID(ctx context.Context, documentID int64) ([]byte, string, string, error) {
	// 1. Get document metadata by ID
	doc, err := uc.repo.GetDocumentByID(ctx, documentID)
	if err != nil {
		return nil, "", "", fmt.Errorf("UseCase - ViewDocumentByID - repo.GetDocumentByID: %w", err)
	}

	// 2. Get file content from S3
	content, err := uc.s3.GetFile(ctx, doc.StoragePath)
	if err != nil {
		return nil, "", "", fmt.Errorf("UseCase - ViewDocumentByID - s3.GetFile: %w", err)
	}

	// Determine content type
	contentType := "application/pdf"
	if doc.FileType == "passport" || doc.FileType == "diploma" {
		contentType = "application/pdf"
	}
	if len(doc.FileName) > 4 && doc.FileName[len(doc.FileName)-4:] == ".jpg" {
		contentType = "image/jpeg"
	} else if len(doc.FileName) > 4 && doc.FileName[len(doc.FileName)-4:] == ".png" {
		contentType = "image/png"
	}

	return content, contentType, doc.FileName, nil
}

func (uc *ApplicantUseCase) GetDocumentStatus(ctx context.Context, documentID int64) (string, error) {
	doc, err := uc.repo.GetDocumentByID(ctx, documentID)
	if err != nil {
		return "", fmt.Errorf("UseCase - GetDocumentStatus - repo.GetDocumentByID: %w", err)
	}
	return doc.Status, nil
}

// Helper functions for map[string]interface{}
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func getStringWithFallback(m map[string]interface{}, key string, fallback string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return fallback
}

func (uc *ApplicantUseCase) SaveExpertEvaluation(ctx context.Context, applicantID int64, expertID int64, userID int64, userName string, role string, category string, score int, comment string) error {
	// userID - who is performing the action
	// expertID - for which expert slot we are saving/overriding
	
	isAdmin := role == "admin"
	
	// If it's an expert, they can only save for themselves
	if !isAdmin && expertID != userID {
		return fmt.Errorf("experts can only submit their own evaluations")
	}

	// 1. Check if the target expertID is actually an assigned expert
	_, err := uc.repo.GetExpertSlotByUserID(ctx, expertID)
	if err != nil && !isAdmin {
		return fmt.Errorf("target user is not an assigned expert")
	}

	// 2. Determine source info
	var sourceInfo string
	if isAdmin {
		sourceInfo = fmt.Sprintf("админ (%s)", userName)
	} else {
		sourceInfo = fmt.Sprintf("эксперт (%s)", userName)
	}

	// 3. Try to get existing evaluation
	eval, err := uc.repo.GetEvaluation(ctx, applicantID, expertID, category)
	if err == nil {
		// Update existing
		eval.Score = score
		eval.Comment = comment
		eval.UpdatedByID = userID
		eval.IsAdminOverride = isAdmin
		eval.SourceInfo = sourceInfo
		
		return uc.repo.UpdateEvaluation(ctx, eval)
	}

	// 4. Create new if not exists
	newEval := entity.ExpertEvaluation{
		ApplicantID:     applicantID,
		ExpertID:        expertID,
		Category:        category,
		Score:           score,
		Comment:         comment,
		UpdatedByID:     userID,
		IsAdminOverride: isAdmin,
		SourceInfo:      sourceInfo,
	}

	return uc.repo.StoreEvaluation(ctx, newEval)
}

func (uc *ApplicantUseCase) ListExpertEvaluations(ctx context.Context, applicantID int64) ([]entity.ExpertEvaluation, error) {
	return uc.repo.ListEvaluations(ctx, applicantID)
}

func (uc *ApplicantUseCase) AssignExpertSlot(ctx context.Context, userID int64, slotNumber int, requesterRole string) error {
	if requesterRole != "admin" {
		return fmt.Errorf("only admins can assign expert slots")
	}

	// Double check max experts limit
	slots, err := uc.repo.GetExpertSlots(ctx)
	if err == nil && len(slots) >= 3 {
		// Check if we are updating an existing slot
		exists := false
		for _, s := range slots {
			if s.SlotNumber == slotNumber {
				exists = true
				break
			}
		}
		if !exists {
			return fmt.Errorf("cannot have more than 3 active experts in the system")
		}
	}

	return uc.repo.AssignExpertSlot(ctx, userID, slotNumber)
}

func (uc *ApplicantUseCase) GetExpertSlots(ctx context.Context) ([]entity.ExpertSlot, error) {
	return uc.repo.GetExpertSlots(ctx)
}

func getInt(m map[string]interface{}, key string, def int) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return def
}

func getFloat(m map[string]interface{}, key string, def float64) float64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
	}
	return def
}
