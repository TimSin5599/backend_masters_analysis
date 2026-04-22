package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"manage-service/internal/domain/entity"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type DocumentUseCase struct {
	repo      DocumentRepo
	appRepo   ApplicantRepo
	queue     DocumentQueueRepo
	producer  DocumentQueueProducer
	extractor ExtractionClient
	s3        S3Provider
}

func NewDocumentUseCase(repo DocumentRepo, appRepo ApplicantRepo, queue DocumentQueueRepo, producer DocumentQueueProducer, extractor ExtractionClient, s3 S3Provider) *DocumentUseCase {
	return &DocumentUseCase{repo: repo, appRepo: appRepo, queue: queue, producer: producer, extractor: extractor, s3: s3}
}

func (uc *DocumentUseCase) GetDocuments(ctx context.Context, applicantID int64) ([]entity.Document, error) {
	return uc.repo.GetDocuments(ctx, applicantID)
}

func (uc *DocumentUseCase) UploadDocument(ctx context.Context, applicantID int64, category string, fileName string, content []byte, docType string) (entity.Document, error) {
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
		Status:      entity.DocStatusPending,
	}

	err = uc.repo.StoreDocument(ctx, &doc)
	if err != nil {
		return entity.Document{}, fmt.Errorf("UseCase - UploadDocument - uc.repo.StoreDocument: %w", err)
	}

	// Update applicant status to 'processing'
	_ = uc.appRepo.Update(ctx, entity.Applicant{ID: applicantID, Status: entity.ApplicantStatusProcessing})


	// 3. Добавление задачи в очередь (RabbitMQ & PostgreSQL)
	priority := 5
	lowerName := strings.ToLower(fileName)
	
	switch category {
	case "passport":
		priority = 10
	case "diploma", "transcript", "second_diploma":
		priority = 9
	case "resume", "experience":
		priority = 8
	case "language", "certification":
		priority = 7
	case "recommendation":
		priority = 6
	case "achievement":
		priority = 5
	case "motivation":
		priority = 4
	case "unknown":
		priority = 1 // Самый низкий для нераспознанных
		// Но если в имени файла есть намек на паспорт - повышаем
		if strings.Contains(lowerName, "pass") || strings.Contains(lowerName, "id") || strings.Contains(lowerName, "passport") {
			priority = 10
		} else if strings.Contains(lowerName, "diploma") || strings.Contains(lowerName, "degree") {
			priority = 9
		} else if strings.Contains(lowerName, "resume") || strings.Contains(lowerName, "cv") {
			priority = 8
		}
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

func (uc *DocumentUseCase) ReprocessDocument(ctx context.Context, documentID int64) error {
	// 1. Get document metadata
	doc, err := uc.repo.GetDocumentByID(ctx, documentID)
	if err != nil {
		return fmt.Errorf("UseCase - ReprocessDocument - repo.GetDocumentByID: %w", err)
	}

	// 2. Reset status to pending (will be picked up by worker again)
	err = uc.repo.UpdateDocumentStatus(ctx, doc.ID, entity.DocStatusPending)
	if err != nil {
		return fmt.Errorf("UseCase - ReprocessDocument - repo.UpdateDocumentStatus: %w", err)
	}

	// Update applicant status to 'processing'
	_ = uc.appRepo.Update(ctx, entity.Applicant{ID: doc.ApplicantID, Status: entity.ApplicantStatusProcessing})


	// 3. Добавление задачи в очередь (RabbitMQ & PostgreSQL)
	priority := 10 // Повторное сканирование вручную всегда получает высокий приоритет
	
	// Если это перенос из unknown, сохраняем высокий приоритет
	if doc.FileType == "unknown" {
		priority = 10
	}

	taskCategory := doc.FileType
	// Если это профессиональное развитие, попробуем восстановить тип из существующих записей
	if doc.FileType == "prof_development" {
		records, _ := uc.appRepo.ListWorkExperience(ctx, doc.ApplicantID, doc.FileType)
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

func (uc *DocumentUseCase) ReprocessLatestDocument(ctx context.Context, applicantID int64, category string) (int64, error) {
	// 1. Get latest document
	doc, err := uc.repo.GetLatestDocumentByCategory(ctx, applicantID, category)
	if err != nil {
		return 0, fmt.Errorf("UseCase - ReprocessLatestDocument - repo.GetLatestDocumentByCategory: %w", err)
	}

	// 2. Delegate to generic reprocess
	return doc.ID, uc.ReprocessDocument(ctx, doc.ID)
}

func (uc *DocumentUseCase) ProcessAIResult(ctx context.Context, applicantID int64, documentID int64, taskCategory string, rawData map[string]string) error {
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
	// For diploma we also clear transcript since extraction always runs both.
	_ = uc.appRepo.DeleteDataByDocumentID(ctx, category, documentID)
	if category == "diploma" {
		_ = uc.appRepo.DeleteDataByDocumentID(ctx, "transcript", documentID)
	}

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

		// Preserve existing name/surname — do not overwrite operator-entered values
		existing, _ := uc.appRepo.GetLatestIdentification(ctx, applicantID)
		name := rawData["name"]
		if name == "" {
			name = existing.Name
		}
		surname := rawData["surname"]
		if surname == "" {
			surname = existing.Surname
		}

		data := entity.IdentificationData{
			ApplicantID:    applicantID,
			DocumentID:     docIDRef,
			Name:           name,
			Surname:        surname,
			Patronymic:     rawData["patronymic"],
			Email:          rawData["email"],
			Phone:          rawData["phone"],
			DocumentNumber: rawData["document_number"],
			Gender:         rawData["gender"],
			Nationality:    rawData["nationality"],
			DateOfBirth:    dob,
			Source:         "model",
		}
		if err := uc.appRepo.StoreIdentification(ctx, data); err != nil {
			return err
		}

	case "achievement", "certification":
		docIDRef := &documentID
		dateReceived := parseTimeSafe(rawData["date_received"])

		data := entity.AchievementData{
			ApplicantID:      applicantID,
			DocumentID:       docIDRef,
			AchievementTitle: rawData["achievement_title"],
			Description:      rawData["description"],
			DateReceived:     dateReceived,
			AchievementType:  rawData["achievement_type"],
			Source:           "model",
		}
		if err := uc.appRepo.StoreAchievement(ctx, data); err != nil {
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
		}
		if err := uc.appRepo.StoreRecommendation(ctx, data); err != nil {
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
		}
		if err := uc.appRepo.StoreEducation(ctx, data); err != nil {
			return err
		}

		// Diploma extraction always runs transcript sub-extraction too.
		// Store transcript data if the document contained a grades table.
		if category == "diploma" {
			cumulativeGPA := parseFloatSafe(rawData["cumulative_gpa"])
			cumulativeGrade := rawData["cumulative_grade"]
			totalCredits := parseFloatSafe(rawData["total_credits"])
			obtainedCredits := parseFloatSafe(rawData["obtained_credits"])
			totalSemesters := parseIntSafe(rawData["total_semesters"])

			if cumulativeGPA > 0 || totalSemesters > 0 {
				existing, err := uc.appRepo.GetTranscriptByDocumentID(ctx, documentID)
				if err == nil {
					existing.CumulativeGPA = cumulativeGPA
					existing.CumulativeGrade = cumulativeGrade
					existing.TotalCredits = totalCredits
					existing.ObtainedCredits = obtainedCredits
					existing.TotalSemesters = totalSemesters
					existing.Source = "model"
					_ = uc.appRepo.UpdateTranscript(ctx, existing)
				} else {
					_ = uc.appRepo.StoreTranscript(ctx, entity.TranscriptData{
						ApplicantID:     applicantID,
						DocumentID:      docIDRef,
						CumulativeGPA:   cumulativeGPA,
						CumulativeGrade: cumulativeGrade,
						TotalCredits:    totalCredits,
						ObtainedCredits: obtainedCredits,
						TotalSemesters:  totalSemesters,
						Source:          "model",
					})
				}
			}
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
		}
		if err := uc.appRepo.StoreLanguageTraining(ctx, data); err != nil {
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
		}
		if err := uc.appRepo.StoreMotivation(ctx, data); err != nil {
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
					RecordType:  recordType,
					Source:      "model",
				}
				if err := uc.appRepo.StoreWorkExperience(ctx, wExp); err != nil {
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
			latest, err := uc.appRepo.GetLatestIdentification(ctx, applicantID)
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
					latest.DocumentID = docIDRef
					_ = uc.appRepo.UpdateIdentification(ctx, latest)
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

				if err := uc.appRepo.StoreIdentification(ctx, entity.IdentificationData{
					ApplicantID: applicantID,
					Email:       newEmail,
					Phone:       finalPhone,
					Source:      "model",
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

			existingExp, _ := uc.appRepo.ListWorkExperience(ctx, applicantID, "")

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
					}
					if err := uc.appRepo.StoreWorkExperience(ctx, wExp); err != nil {
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

			existingAch, _ := uc.appRepo.ListAchievements(ctx, applicantID, "")

			for _, ach := range newAchievements {
				title, _ := ach["achievement_title"].(string)
				dateStr, _ := ach["date_received"].(string)

				receivedDate := parseTimeSafe(dateStr)

				isDuplicate := false
				cleanNewTitle := normalizeForDedupe(title)

				for _, ea := range existingAch {
					if normalizeForDedupe(ea.AchievementTitle) == cleanNewTitle &&
						(ea.DateReceived.Equal(receivedDate) || (ea.DateReceived.Year() == receivedDate.Year() && ea.DateReceived.Month() == receivedDate.Month())) {
						isDuplicate = true
						break
					}
				}

				if !isDuplicate {
					aData := entity.AchievementData{
						ApplicantID:      applicantID,
						DocumentID:       docIDRef,
						AchievementTitle: title,
						DateReceived:     receivedDate,
						Source:           "model",
					}
					if err := uc.appRepo.StoreAchievement(ctx, aData); err != nil {
						fmt.Printf("[USECASE] ❌ Error storing achievement: %v\n", err)
					} else {
						fmt.Printf("[USECASE] ✅ Stored achievement: %s\n", title)
					}
				} else {
					fmt.Printf("[USECASE] ⏭️ Skipped duplicate achievement: %s (Date: %s)\n", title, dateStr)
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

		existing, err := uc.appRepo.GetTranscriptByDocumentID(ctx, documentID)
		if err == nil {
			// UPDATE existing
			existing.CumulativeGPA = cumulativeGPA
			existing.CumulativeGrade = cumulativeGrade
			existing.TotalCredits = totalCredits
			existing.ObtainedCredits = obtainedCredits
			existing.TotalSemesters = totalSemesters
			existing.Source = "model"
			err = uc.appRepo.UpdateTranscript(ctx, existing)
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
			}
			err = uc.appRepo.StoreTranscript(ctx, data)
		}
		if err != nil {
			return err
		}
	}

	// If we reached here, data was stored successfully. Update document status.
	return uc.repo.UpdateDocumentStatus(ctx, documentID, "completed")
}

func (uc *DocumentUseCase) DeleteDocument(ctx context.Context, applicantID int64, documentID int64) error {
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
		err = uc.appRepo.DeleteDataByDocumentID(ctx, cat, documentID)
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

func (uc *DocumentUseCase) ViewDocument(ctx context.Context, applicantID int64, category string) ([]byte, string, string, error) {
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

func (uc *DocumentUseCase) ViewDocumentByID(ctx context.Context, documentID int64) ([]byte, string, string, error) {
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

func (uc *DocumentUseCase) GetDocumentStatus(ctx context.Context, documentID int64) (string, error) {
	doc, err := uc.repo.GetDocumentByID(ctx, documentID)
	if err != nil {
		return "", fmt.Errorf("UseCase - GetDocumentStatus - repo.GetDocumentByID: %w", err)
	}
	return doc.Status, nil
}

// UpdateDocumentStatus позволяет вручную обновить статус документа.
// Используется после ручного ввода данных оператором (extraction_failed → completed).
func (uc *DocumentUseCase) UpdateDocumentStatus(ctx context.Context, documentID int64, status string) error {
	return uc.repo.UpdateDocumentStatus(ctx, documentID, status)
}

func (uc *DocumentUseCase) GetQueueTasks(ctx context.Context, applicantID int64) ([]entity.DocumentQueueTask, error) {
	return uc.queue.GetByApplicantID(ctx, applicantID)
}

func (uc *DocumentUseCase) ChangeDocumentCategory(ctx context.Context, documentID int64, newCategory string) error {
	// 0. Read the document FIRST to capture the old category before overwriting it
	doc, err := uc.repo.GetDocumentByID(ctx, documentID)
	if err != nil {
		return fmt.Errorf("UseCase - ChangeDocumentCategory - repo.GetDocumentByID: %w", err)
	}
	oldCategory := doc.FileType

	// 1. Update the category in DB
	err = uc.repo.UpdateDocumentCategory(ctx, documentID, newCategory)
	if err != nil {
		return fmt.Errorf("UseCase - ChangeDocumentCategory - repo.UpdateDocumentCategory: %w", err)
	}

	// 2. Delete extracted data that belonged to the old category so reprocessing starts clean.
	//    Ignore errors — the data may not exist yet if the document was never fully processed.
	if oldCategory != newCategory {
		_ = uc.appRepo.DeleteDataByDocumentID(ctx, oldCategory, documentID)
		// Diploma documents also produce transcript rows — clean those up too.
		if oldCategory == "diploma" {
			_ = uc.appRepo.DeleteDataByDocumentID(ctx, "transcript", documentID)
		}
	}

	// 3. Trigger reprocessing so AI extracts data according to the new category
	return uc.ReprocessDocument(ctx, documentID)
}
