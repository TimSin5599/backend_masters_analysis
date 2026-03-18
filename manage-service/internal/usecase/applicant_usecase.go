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

func (uc *ApplicantUseCase) GetDocuments(ctx context.Context, applicantID int64) ([]entity.Document, error) {
	return uc.repo.GetDocuments(ctx, applicantID)
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
	case "second_diploma":
		return uc.repo.ListEducation(ctx, applicantID)
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
func (uc *ApplicantUseCase) UploadDocument(ctx context.Context, applicantID int64, category string, fileName string, content []byte, docType string) (entity.Document, error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
		pdfBytes, err := convertImageToPDF(fileName, content)
		if err != nil {
			return entity.Document{}, fmt.Errorf("failed to convert image to pdf: %w", err)
		}
		content = pdfBytes
		fileName = fileName[:len(fileName)-len(ext)] + ".pdf"
	}

	// 1. Сохранение в MinIO
	storagePath := fmt.Sprintf("applicants/%d/%s/%s", applicantID, category, fileName)
	err := uc.s3.UploadFile(ctx, storagePath, content)
	if err != nil {
		return entity.Document{}, fmt.Errorf("UseCase - UploadDocument - s3.UploadFile: %w", err)
	}

	// 2. Регистрация в БД
	fileType := category
	if docType != "" {
		fileType = docType
	}
	doc := entity.Document{
		ApplicantID: applicantID,
		FileType:    fileType,
		StoragePath: storagePath,
		FileName:    fileName,
		Status:      "processing",
	}

	err = uc.repo.StoreDocument(ctx, &doc)
	if err != nil {
		return entity.Document{}, fmt.Errorf("UseCase - UploadDocument - uc.repo.StoreDocument: %w", err)
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
	case "experience", "resume":
		priority = 7
	case "recommendation":
		priority = 6
	case "achievement":
		priority = 5
	case "motivation":
		priority = 4
	}

	taskCategory := category
	if docType != "" {
		taskCategory = fmt.Sprintf("%s:%s", category, docType)
	}

	task := entity.DocumentQueueTask{
		ApplicantID:      applicantID,
		DocumentCategory: taskCategory,
		FilePath:         storagePath,
		Priority:         priority,
		Status:           "pending",
	}

	queueID, err := uc.queue.Enqueue(ctx, task)
	if err != nil {
		return entity.Document{}, fmt.Errorf("UseCase - UploadDocument - queue.Enqueue: %w", err)
	}

	task.ID = queueID
	err = uc.producer.PublishTask(task)
	if err != nil {
		// If publishing fails, mark as failed in DB
		errMsg := err.Error()
		_ = uc.queue.UpdateStatus(context.Background(), queueID, "failed", &errMsg)
		return entity.Document{}, fmt.Errorf("UseCase - UploadDocument - producer.PublishTask: %w", err)
	}

	return doc, nil
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

	taskCategory := doc.FileType
	// Если это профессиональное развитие, попробуем восстановить тип из существующих записей
	if doc.FileType == "prof_development" {
		records, _ := uc.repo.ListWorkExperience(ctx, doc.ApplicantID, doc.FileType)
		for _, r := range records {
			if r.DocumentID != nil && *r.DocumentID == documentID && r.RecordType != "" {
				taskCategory = fmt.Sprintf("%s:%s", doc.FileType, r.RecordType)
				break
			}
		}
	}

	task := entity.DocumentQueueTask{
		ApplicantID:      doc.ApplicantID,
		DocumentCategory: taskCategory,
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
func (uc *ApplicantUseCase) ProcessAIResult(ctx context.Context, applicantID int64, documentID int64, taskCategory string, rawData map[string]string) error {
	category := taskCategory
	docType := ""
	if strings.Contains(taskCategory, ":") {
		parts := strings.SplitN(taskCategory, ":", 2)
		category = parts[0]
		docType = parts[1]
	}

	fmt.Printf("[USECASE] ProcessAIResult: Category=%s, Applicant=%d, Doc=%d, Keys=%v\n", category, applicantID, documentID, getKeys(rawData))
	// 0. Check for AI extraction errors
	if errStr, ok := rawData["error"]; ok {
		return fmt.Errorf("AI extraction failed: %s", errStr)
	}

	// 1. Сохраняем в лог extracted_fields
	for k, v := range rawData {
		_ = uc.repo.StoreExtractedField(ctx, applicantID, documentID, k, v)
	}

	// 2. В зависимости от категории наполняем конкретную таблицу (Version 1, source='model')
	
	// Prioritize resume processing if docType is resume
	if docType == "resume" {
		category = "resume"
	}

	normalizeForDedupe := func(s string) string {
		s = strings.ToLower(s)
		// Remove common punctuation and trim
		s = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r >= 'а' && r <= 'я') || r == ' ' {
				return r
			}
			return -1
		}, s)
		return strings.Join(strings.Fields(s), " ") // Remove extra spaces
	}
	parseTimeSafe := func(s string) time.Time {
		if s == "" {
			return time.Time{}
		}
		// Some models return date strings inconsistently.
		// Try a few formats: ISO, Russian, English textual
		formats := []string{"2006-01-02", "02.01.2006", "02 Jan 2006", "2 Jan 2006"}
		for _, f := range formats {
			if t, e := time.Parse(f, s); e == nil {
				return t
			}
		}
		// Try to handle "10 OCT 1995" if it comes in uppercase or variations
		s_cl := strings.Title(strings.ToLower(s))
		if t, e := time.Parse("02 Jan 2006", s_cl); e == nil {
			return t
		}
		return time.Time{}
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
			Email:          rawData["email"],
			Phone:          rawData["phone"],
			DocumentNumber: rawData["document_number"],
			Gender:         rawData["gender"],
			Nationality:    rawData["nationality"],
			DateOfBirth:    dob,
			Source:         "model",
			Version:        1,
		}
		if err := uc.repo.StoreIdentification(ctx, data); err != nil {
			return err
		}

	case "achievement", "certification":
		docIDRef := &documentID
		dateReceived := parseTimeSafe(rawData["date_received"])
		// AI may use "company" instead of "company_name" after prompt update
		company := rawData["company"]
		if company == "" {
			company = rawData["company_name"]
		}

		data := entity.AchievementData{
			ApplicantID:      applicantID,
			DocumentID:       docIDRef,
			AchievementTitle: rawData["achievement_title"],
			Description:      rawData["description"],
			DateReceived:     dateReceived,
			AchievementType:  rawData["achievement_type"],
			Company:          company,
			Source:           "model",
			Version:          1,
		}
		if err := uc.repo.StoreAchievement(ctx, data); err != nil {
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

	case "diploma", "second_diploma":
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

	case "prof_development", "work":
		recordType := docType

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
					RecordType:  recordType,
					Source:      "model",
					Version:     1,
				}
				if err := uc.repo.StoreWorkExperience(ctx, wExp); err != nil {
					return err
				}
			}
		}

	case "resume":
		docIDRef := &documentID
		
		// 1. Identification (Email, Phone) - Merge with existing
		newEmail := rawData["email"]
		newPhone := rawData["phone"]
		
		if newEmail != "" || newPhone != "" {
			latest, err := uc.repo.GetLatestIdentification(ctx, applicantID)
			if err == nil {
				// Overwrite logic: prefer data from model
				updated := false
				if newEmail != "" {
					latest.Email = newEmail
					updated = true
				}
				
				if newPhone != "" {
					// Normalize new phone
					normalizedNewPhone := strings.ReplaceAll(newPhone, " +", ", +")
					parts := strings.Split(normalizedNewPhone, ",")
					var cleanParts []string
					for _, p := range parts {
						p = strings.TrimSpace(p)
						if p != "" {
							cleanParts = append(cleanParts, p)
						}
					}
					finalPhone := strings.Join(cleanParts, ", ")
					
					if finalPhone != "" {
						latest.Phone = finalPhone
						updated = true
					}
				}
				
				if updated {
					latest.Source = "model"
					latest.Version = 1
					latest.DocumentID = docIDRef
					_ = uc.repo.UpdateIdentification(ctx, latest)
				}
			} else {
				// No existing record, create new one
				// If incoming phone has multiple numbers but no commas, normalize it
				normalizedPhone := strings.ReplaceAll(newPhone, " +", ", +")
				// Ensure it's clean (trim all parts)
				parts := strings.Split(normalizedPhone, ",")
				var cleanParts []string
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p != "" {
						cleanParts = append(cleanParts, p)
					}
				}
				finalPhone := strings.Join(cleanParts, ", ")

				if err := uc.repo.StoreIdentification(ctx, entity.IdentificationData{
					ApplicantID: applicantID,
					Email:       newEmail,
					Phone:       finalPhone,
					Source:      "model",
					Version:     1,
					DocumentID:  docIDRef,
				}); err != nil {
					fmt.Printf("[USECASE] ❌ Error storing new identification from resume: %v\n", err)
				} else {
					fmt.Printf("[USECASE] ✅ Stored new identification for applicant %d from resume\n", applicantID)
				}
			}
		}

		// 2. Experiences - Deduplicate
		var newExperiences []map[string]interface{}
		if expRaw, ok := rawData["experiences"]; ok && expRaw != "" {
			if err := json.Unmarshal([]byte(expRaw), &newExperiences); err != nil {
				fmt.Printf("[USECASE] ❌ Failed to unmarshal experiences: %v\n", err)
			}
			fmt.Printf("[USECASE] Found %d experiences in resume for applicant %d\n", len(newExperiences), applicantID)
			
			existingExp, _ := uc.repo.ListWorkExperience(ctx, applicantID, "")
			
			for _, exp := range newExperiences {
				companyName, _ := exp["company_name"].(string)
				position, _ := exp["position"].(string)
				startDateStr, _ := exp["start_date"].(string)
				endDateStr, _ := exp["end_date"].(string)
				recordType, _ := exp["record_type"].(string)
				if recordType == "" {
					recordType = "work"
				}
				
				startDate := parseTimeSafe(startDateStr)
				var endDate *time.Time
				if endDateStr != "" {
					et := parseTimeSafe(endDateStr)
					endDate = &et
				}

				// Check for duplicates with normalization
				isDuplicate := false
				cleanNewCompany := normalizeForDedupe(companyName)
				cleanNewPosition := normalizeForDedupe(position)

				for _, ee := range existingExp {
					if normalizeForDedupe(ee.CompanyName) == cleanNewCompany && 
					   normalizeForDedupe(ee.Position) == cleanNewPosition && 
					   (ee.StartDate.Equal(startDate) || (ee.StartDate.Year() == startDate.Year() && ee.StartDate.Month() == startDate.Month())) {
						isDuplicate = true
						break
					}
				}

				if !isDuplicate {
					wExp := entity.WorkExperience{
						ApplicantID: applicantID,
						DocumentID:  docIDRef,
						CompanyName: companyName,
						Position:    position,
						StartDate:   startDate,
						EndDate:     endDate,
						RecordType:  recordType,
						Source:      "model",
						Version:     1,
					}
					if err := uc.repo.StoreWorkExperience(ctx, wExp); err != nil {
						fmt.Printf("[USECASE] ❌ Error storing experience: %v\n", err)
					} else {
						fmt.Printf("[USECASE] ✅ Stored experience: %s at %s\n", position, companyName)
					}
				} else {
					fmt.Printf("[USECASE] ⏭️ Skipped duplicate experience: %s at %s (Starts: %s)\n", position, companyName, startDateStr)
				}
			}
		}

		// 3. Achievements - Deduplicate
		var newAchievements []map[string]interface{}
		if achRaw, ok := rawData["achievements"]; ok && achRaw != "" {
			if err := json.Unmarshal([]byte(achRaw), &newAchievements); err != nil {
				fmt.Printf("[USECASE] ❌ Failed to unmarshal achievements: %v\n", err)
			}
			fmt.Printf("[USECASE] Found %d achievements in resume for applicant %d\n", len(newAchievements), applicantID)

			existingAch, _ := uc.repo.ListAchievements(ctx, applicantID, "")
			
			for _, ach := range newAchievements {
				title, _ := ach["achievement_title"].(string)
				company, _ := ach["company_name"].(string)
				dateStr, _ := ach["date_received"].(string)
				
				receivedDate := parseTimeSafe(dateStr)

				isDuplicate := false
				cleanNewTitle := normalizeForDedupe(title)
				cleanNewCompany := normalizeForDedupe(company)

				for _, ea := range existingAch {
					if normalizeForDedupe(ea.AchievementTitle) == cleanNewTitle && 
					   normalizeForDedupe(ea.Company) == cleanNewCompany && 
					   (ea.DateReceived.Equal(receivedDate) || (ea.DateReceived.Year() == receivedDate.Year() && ea.DateReceived.Month() == receivedDate.Month())) {
						isDuplicate = true
						break
					}
				}

				if !isDuplicate {
					aData := entity.AchievementData{
						ApplicantID:      applicantID,
						DocumentID:       docIDRef,
						AchievementType:  "", // Placeholder or extract if needed
						AchievementTitle: title,
						Company:          company,
						DateReceived:     receivedDate,
						Source:           "model",
						Version:          1,
					}
					if err := uc.repo.StoreAchievement(ctx, aData); err != nil {
						fmt.Printf("[USECASE] ❌ Error storing achievement: %v\n", err)
					} else {
						fmt.Printf("[USECASE] ✅ Stored achievement: %s from %s\n", title, company)
					}
				} else {
					fmt.Printf("[USECASE] ⏭️ Skipped duplicate achievement: %s from %s (Date: %s)\n", title, company, dateStr)
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

func (uc *ApplicantUseCase) DeleteDocument(ctx context.Context, applicantID int64, documentID int64) error {
	// 1. Get document metadata to get storage path
	doc, err := uc.repo.GetDocumentByID(ctx, documentID)
	if err != nil {
		return fmt.Errorf("UseCase - DeleteDocument - repo.GetDocumentByID: %w", err)
	}

	if doc.ApplicantID != applicantID {
		return fmt.Errorf("UseCase - DeleteDocument - document %d does not belong to applicant %d", documentID, applicantID)
	}

	// 2. Remove from MinIO
	err = uc.s3.DeleteFile(ctx, doc.StoragePath)
	if err != nil {
		fmt.Printf("[USECASE] Warning: failed to delete file from S3: %v\n", err)
		// We continue to delete from DB even if S3 fails (orphan file is better than stale data)
	}

	// 3. Delete extracted data from all tables
	// We call it multiple times for different categories that might be associated with this document
	categories := []string{"passport", "resume", "diploma", "education", "recommendation", "achievement", "language", "motivation", "work", "prof_development", "transcript", "second_diploma", "certification"}
	for _, cat := range categories {
		err = uc.repo.DeleteDataByDocumentID(ctx, cat, documentID)
		if err != nil {
			fmt.Printf("[USECASE] Warning: failed to delete extracted data for category %s: %v\n", cat, err)
		}
	}

	// 4. Delete the document record itself
	err = uc.repo.DeleteDocument(ctx, documentID)
	if err != nil {
		return fmt.Errorf("UseCase - DeleteDocument - repo.DeleteDocument: %w", err)
	}

	fmt.Printf("[USECASE] ✅ Document %d and its data deleted for applicant %d\n", documentID, applicantID)
	return nil
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
	case "passport", "resume":
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

	case "work", "prof_development", "recommendation", "achievement":
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
					Company:          getString(m, "company_name"),
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

func (uc *ApplicantUseCase) GetEvaluationCriteria(ctx context.Context) ([]entity.EvaluationCriteria, error) {
	return uc.repo.GetCriteria(ctx)
}

func (uc *ApplicantUseCase) SaveExpertEvaluation(ctx context.Context, applicantID int64, expertID string, userID string, userName string, role string, evaluations []entity.ExpertEvaluation, complete bool) error {
	isAdmin := role == "admin"
	
	// 1. If it's an expert, they can only save for themselves
	if !isAdmin && expertID != userID {
		return fmt.Errorf("experts can only submit their own evaluations")
	}

	// 2. Check if the target expertID is actually an assigned expert
	_, err := uc.repo.GetExpertSlotByUserID(ctx, expertID)
	if err != nil && !isAdmin {
		return fmt.Errorf("target user is not an assigned expert")
	}

	// 3. Check for immutability: if any existing score is COMPLETED
	existingEvals, err := uc.repo.ListEvaluations(ctx, applicantID)
	if err == nil {
		for _, e := range existingEvals {
			if e.ExpertID == expertID && e.Status == entity.EvaluationStatusCompleted && !isAdmin {
				return entity.ErrEvaluationImmutable
			}
		}
	}

	// 4. Validate scores against criteria
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

	// 5. Prepare batch
	toSave := make([]entity.ExpertEvaluation, 0, len(evaluations))
	for _, e := range evaluations {
		c, ok := criteriaMap[e.Category]
		if !ok {
			return fmt.Errorf("unknown criteria: %s", e.Category)
		}
		if e.Score > c.MaxScore || e.Score < 0 {
			return entity.ErrScoreExceedsMax
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

	// 6. Save batch
	if err := uc.repo.SaveEvaluationBatch(ctx, toSave); err != nil {
		return err
	}

	// 7. If completed, trigger aggregation
	if complete {
		return uc.triggerAggregation(ctx, applicantID)
	}

	return nil
}

func (uc *ApplicantUseCase) triggerAggregation(ctx context.Context, applicantID int64) error {
	// Check if all 3 experts (slots) have completed their evaluations
	slots, err := uc.repo.GetExpertSlots(ctx)
	if err != nil {
		return err
	}

	evals, err := uc.repo.ListEvaluations(ctx, applicantID)
	if err != nil {
		return err
	}

	completedExperts := make(map[string]bool)
	for _, e := range evals {
		if e.Status == entity.EvaluationStatusCompleted {
			completedExperts[e.ExpertID] = true
		}
	}

	assignedExpertsCount := 0
	for _, s := range slots {
		if s.UserID != "" {
			assignedExpertsCount++
		}
	}

	// If all assigned experts (max 3) are done, or at least 1 if only 1 assigned (simplified)
	// The requirement says "3 independent experts", so we should check for completion of all assigned.
	// We'll proceed if all assigned slots are completed.
	allDone := true
	for _, s := range slots {
		if s.UserID != "" && !completedExperts[s.UserID] {
			allDone = false
			break
		}
	}

	if allDone && assignedExpertsCount > 0 {
		score, err := uc.repo.GetAggregatedScore(ctx, applicantID)
		if err != nil {
			return err
		}
		return uc.repo.UpdateApplicantRanking(ctx, applicantID, score, "EVALUATED")
	}

	return uc.repo.UpdateApplicantRanking(ctx, applicantID, 0, "IN_PROGRESS")
}

func (uc *ApplicantUseCase) ListExpertEvaluations(ctx context.Context, applicantID int64, currentUserID string) ([]entity.ExpertEvaluation, error) {
	evals, err := uc.repo.ListEvaluations(ctx, applicantID)
	if err != nil {
		return nil, err
	}

	// "Blind Grading" logic:
	// If the current user (expert) hasn't completed their evaluation,
	// mask scores of other experts.
	
	currentUserCompleted := false
	for _, e := range evals {
		if e.ExpertID == currentUserID && e.Status == entity.EvaluationStatusCompleted {
			currentUserCompleted = true
			break
		}
	}

	// If not completed, mask others
	if !currentUserCompleted {
		for i := range evals {
			if evals[i].ExpertID != currentUserID {
				evals[i].Score = -1 // Masked
				evals[i].Comment = "Score hidden until you complete your evaluation"
			}
		}
	}

	return evals, nil
}

func (uc *ApplicantUseCase) AssignExpertSlot(ctx context.Context, userID string, slotNumber int, requesterRole string) error {
	if requesterRole != "admin" && requesterRole != "manager" {
		return fmt.Errorf("only admins and managers can assign expert slots")
	}

	fmt.Printf("[DEBUG] AssignExpertSlot: userID=%s, slotNumber=%d, role=%s\n", userID, slotNumber, requesterRole)
	if userID == "" || userID == "0" {
		return uc.repo.RemoveExpertSlot(ctx, slotNumber)
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

func (uc *ApplicantUseCase) ListExperts(ctx context.Context) ([]entity.User, error) {
	return uc.repo.GetUsersByRoles(ctx, []string{"expert"})
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

func getKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
