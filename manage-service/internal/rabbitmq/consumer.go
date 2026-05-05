package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"manage-service/internal/domain/entity"
	"manage-service/internal/sse"
	"manage-service/internal/usecase"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	url        string
	conn       *amqp.Connection
	ch         *amqp.Channel
	repo       usecase.ApplicantRepo
	queueRepo  usecase.DocumentQueueRepo
	appUseCase usecase.Applicant
	docUseCase usecase.Document
	docRepo    usecase.DocumentRepo
	s3         usecase.S3Provider
	extractor  usecase.ExtractionClient
	hub        *sse.Hub
}

func NewConsumer(url string, repo usecase.ApplicantRepo, queueRepo usecase.DocumentQueueRepo, appUseCase usecase.Applicant, docUseCase usecase.Document, docRepo usecase.DocumentRepo, s3 usecase.S3Provider, extractor usecase.ExtractionClient, hub *sse.Hub) (*Consumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	if err := declareQueue(ch); err != nil {
		return nil, err
	}

	// Fetch 1 message at a time
	err = ch.Qos(1, 0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return &Consumer{
		url:        url,
		conn:       conn,
		ch:         ch,
		repo:       repo,
		queueRepo:  queueRepo,
		appUseCase: appUseCase,
		docUseCase: docUseCase,
		docRepo:    docRepo,
		s3:         s3,
		extractor:  extractor,
		hub:        hub,
	}, nil
}

// declareQueue declares the queue with required arguments. Called on init and after reconnect.
func declareQueue(ch *amqp.Channel) error {
	args := amqp.Table{
		"x-max-priority": int32(10),
	}
	_, err := ch.QueueDeclare(
		QueueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		args,
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}
	return nil
}

func (c *Consumer) Close() {
	if c.ch != nil {
		c.ch.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	for {
		msgs, err := c.ch.Consume(
			QueueName,
			"",
			false, // auto-ack
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Printf(" [!] Failed to register a consumer: %v. Retrying in 5s...", err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(5 * time.Second):
				c.reconnect()
				continue
			}
		}

		log.Println(" [*] RabbitMQ Consumer started. Waiting for messages.")

		errChan := make(chan *amqp.Error, 1)
		c.ch.NotifyClose(errChan)

		shouldReconnect := false
	loop:
		for {
			select {
			case <-ctx.Done():
				log.Println(" [*] Shutting down RabbitMQ Consumer.")
				return nil
			case err := <-errChan:
				log.Printf(" [!] RabbitMQ channel closed: %v. Reconnecting...", err)
				shouldReconnect = true
				break loop
			case d, ok := <-msgs:
				if !ok {
					log.Println(" [!] RabbitMQ messages channel closed.")
					shouldReconnect = true
					break loop
				}
				c.processMessage(d)
			}
		}

		if shouldReconnect {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(5 * time.Second):
				c.reconnect()
				continue
			}
		}
	}
}

func (c *Consumer) reconnect() {
	for {
		log.Println(" [*] Attempting to reconnect to RabbitMQ...")
		conn, err := amqp.Dial(c.url)
		if err != nil {
			log.Printf(" [!] Reconnection failed: %v. Retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		ch, err := conn.Channel()
		if err != nil {
			log.Printf(" [!] Failed to open channel: %v. Retrying in 5s...", err)
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		// Re-declare queue after reconnect — required if broker restarted or queue was lost
		if err := declareQueue(ch); err != nil {
			log.Printf(" [!] Failed to declare queue after reconnect: %v. Retrying in 5s...", err)
			ch.Close()
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		if err := ch.Qos(1, 0, false); err != nil {
			log.Printf(" [!] Failed to set QoS after reconnect: %v. Retrying in 5s...", err)
			ch.Close()
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		c.conn = conn
		c.ch = ch
		log.Println(" [*] Successfully reconnected to RabbitMQ!")
		return
	}
}

func (c *Consumer) processMessage(d amqp.Delivery) {
	// Recover from any panic so the consumer goroutine stays alive
	defer func() {
		if r := recover(); r != nil {
			log.Printf(" [!] Panic in processMessage: %v", r)
			// Requeue only on first delivery to avoid infinite panic loop
			d.Nack(false, !d.Redelivered)
		}
	}()

	var task entity.DocumentQueueTask
	if err := json.Unmarshal(d.Body, &task); err != nil {
		log.Printf(" [!] Failed to unmarshal message body: %v", err)
		d.Nack(false, false) // Malformed message — discard
		return
	}

	log.Printf(" [->] Processing Document %s (Priority: %d, Category: %s)", task.ID, task.Priority, task.DocumentCategory)

	// Hard outer deadline: classification (2m) + extraction (up to 11m for diploma) + buffer
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	_ = c.queueRepo.UpdateStatus(ctx, task.ID, "processing", nil)
	c.hub.BroadcastStatus(task.ApplicantID, map[string]interface{}{
		"task_id":  task.ID,
		"category": task.DocumentCategory,
		"status":   entity.DocStatusPending,
		"progress": 10,
	})

	docs, err := c.docRepo.GetDocuments(ctx, task.ApplicantID)
	if err != nil {
		// DB error is transient — requeue once, no doc ID to update
		c.failTask(task.ApplicantID, task.ID, task.DocumentCategory, 0, entity.DocStatusPending, d,
			fmt.Errorf("failed to get applicant documents: %w", err), !d.Redelivered)
		return
	}

	var targetDoc entity.Document
	found := false
	for _, doc := range docs {
		if doc.StoragePath == task.FilePath {
			targetDoc = doc
			found = true
			break
		}
	}

	if !found {
		// Document deleted — no point retrying
		c.failTask(task.ApplicantID, task.ID, task.DocumentCategory, 0, entity.DocStatusExtractionFailed, d,
			fmt.Errorf("document metadata not found in db for path %s", task.FilePath), false)
		return
	}

	content, err := c.s3.GetFile(ctx, task.FilePath)
	if err != nil {
		// S3 error is transient — requeue once
		c.failTask(task.ApplicantID, task.ID, task.DocumentCategory, targetDoc.ID, entity.DocStatusExtractionFailed, d,
			fmt.Errorf("failed to get file from S3: %w", err), !d.Redelivered)
		return
	}

	// ── Classification stage ──────────────────────────────────────────────────
	if targetDoc.FileType == "unknown" || task.DocumentCategory == "unknown" {
		_ = c.docRepo.UpdateDocumentStatus(ctx, targetDoc.ID, entity.DocStatusClassifying)
		c.hub.BroadcastStatus(task.ApplicantID, map[string]interface{}{
			"task_id":  task.ID,
			"category": task.DocumentCategory,
			"status":   entity.DocStatusClassifying,
			"progress": 35,
		})
		fmt.Printf("[CONSUMER] Triggering classification for applicant %d, file %s\n", task.ApplicantID, targetDoc.FileName)

		predCat, _, err := c.extractor.ClassifyDocument(ctx, targetDoc.FileName, content)
		if err != nil {
			fmt.Printf("[CONSUMER] ❌ Classification failed for applicant %d: %v\n", task.ApplicantID, err)
			// On first attempt requeue; on retry mark as classification_failed so user can pick category manually
			requeue := !d.Redelivered
			if !requeue {
				_ = c.docRepo.UpdateDocumentStatus(ctx, targetDoc.ID, entity.DocStatusClassificationFailed)
			}
			c.failTask(task.ApplicantID, task.ID, task.DocumentCategory, targetDoc.ID, entity.DocStatusClassificationFailed, d,
				fmt.Errorf("AI classification failed: %w", err), requeue)
			return
		}

		sysCategory := predCat
		docType := ""
		switch predCat {
		case "work":
			sysCategory = "prof_development"
			docType = "work"
		case "professional_development":
			sysCategory = "prof_development"
			docType = "training"
		case "certificate":
			sysCategory = "certification"
		case "diploma":
			// If applicant already has a diploma, route to second_diploma
			for _, doc := range docs {
				if doc.FileType == "diploma" && doc.ID != targetDoc.ID {
					sysCategory = "second_diploma"
					break
				}
			}
		}

		targetDoc.FileType = sysCategory
		task.DocumentCategory = sysCategory
		if docType != "" {
			task.DocumentCategory = fmt.Sprintf("%s:%s", sysCategory, docType)
		}

		if err = c.docRepo.UpdateDocumentCategory(ctx, targetDoc.ID, sysCategory); err != nil {
			fmt.Printf("[CONSUMER] Warning: failed to save classified category: %v\n", err)
		}

		// Mark as classified — category is known, extraction will follow
		_ = c.docRepo.UpdateDocumentStatus(ctx, targetDoc.ID, entity.DocStatusClassified)
		c.hub.BroadcastStatus(task.ApplicantID, map[string]interface{}{
			"task_id":  task.ID,
			"category": task.DocumentCategory,
			"status":   entity.DocStatusClassified,
			"progress": 45,
		})
	}

	// If still unknown after classification attempt (model returned unknown), stop and let user decide
	if targetDoc.FileType == "unknown" {
		log.Printf(" [<-] Document %s remains unknown after classification. User must assign category.", task.ID)
		_ = c.docRepo.UpdateDocumentStatus(ctx, targetDoc.ID, entity.DocStatusClassificationFailed)
		_ = c.queueRepo.UpdateStatus(ctx, task.ID, "failed", nil)
		c.hub.BroadcastStatus(task.ApplicantID, map[string]interface{}{
			"task_id":  task.ID,
			"category": task.DocumentCategory,
			"status":   entity.DocStatusClassificationFailed,
			"progress": 0,
		})
		d.Ack(false)
		return
	}

	// ── Extraction stage ──────────────────────────────────────────────────────
	_ = c.docRepo.MarkProcessingStarted(ctx, targetDoc.ID)
	_ = c.docRepo.UpdateDocumentStatus(ctx, targetDoc.ID, entity.DocStatusExtracting)
	c.hub.BroadcastStatus(task.ApplicantID, map[string]interface{}{
		"task_id":  task.ID,
		"category": task.DocumentCategory,
		"status":   entity.DocStatusExtracting,
		"progress": 55,
	})
	fmt.Printf("[CONSUMER] Triggering extraction for applicant %d, category %s, file %s\n", task.ApplicantID, task.DocumentCategory, task.FilePath)
	rawData, err := c.extractor.TriggerExtraction(ctx, targetDoc, content)
	if err != nil {
		fmt.Printf("[CONSUMER] ❌ Extraction failed for applicant %d: %v\n", task.ApplicantID, err)
		requeue := !d.Redelivered
		if !requeue {
			_ = c.docRepo.UpdateDocumentStatus(ctx, targetDoc.ID, entity.DocStatusExtractionFailed)
		}
		c.failTask(task.ApplicantID, task.ID, task.DocumentCategory, targetDoc.ID, entity.DocStatusExtractionFailed, d,
			fmt.Errorf("AI extraction failed: %w", err), requeue)
		return
	}
	fmt.Printf("[CONSUMER] ✅ Extraction success for applicant %d. Raw data keys: %v\n", task.ApplicantID, getKeys(rawData))

	c.hub.BroadcastStatus(task.ApplicantID, map[string]interface{}{
		"task_id":  task.ID,
		"category": task.DocumentCategory,
		"status":   "saving",
		"progress": 80,
	})
	if err = c.docUseCase.ProcessAIResult(ctx, targetDoc.ApplicantID, targetDoc.ID, task.DocumentCategory, rawData); err != nil {
		// ProcessAIResult failure — mark extraction_failed so user can enter manually
		_ = c.docRepo.UpdateDocumentStatus(ctx, targetDoc.ID, entity.DocStatusExtractionFailed)
		c.failTask(task.ApplicantID, task.ID, task.DocumentCategory, targetDoc.ID, entity.DocStatusExtractionFailed, d,
			fmt.Errorf("failed to process AI result: %w", err), !d.Redelivered)
		return
	}

	_ = c.queueRepo.UpdateStatus(ctx, task.ID, "completed", nil)
	log.Printf(" [<-] Completed Document %s", task.ID)
	c.hub.BroadcastStatus(task.ApplicantID, map[string]interface{}{
		"task_id":  task.ID,
		"category": task.DocumentCategory,
		"status":   entity.DocStatusCompleted,
		"progress": 100,
	})
	d.Ack(false)

	// После завершения каждого документа проверяем — все ли готовы
	c.autoTransferIfReady(context.Background(), task.ApplicantID)
}

// autoTransferIfReady переводит абитуриента в статус "verifying" если все документы обработаны.
// Если не хватает обязательных документов — отправляет WS-уведомление менеджеру.
func (c *Consumer) autoTransferIfReady(ctx context.Context, applicantID int64) {
	docs, err := c.docRepo.GetDocuments(ctx, applicantID)
	if err != nil {
		log.Printf("[AUTO-TRANSFER] Failed to get documents for applicant %d: %v", applicantID, err)
		return
	}

	// Wait for all in-flight statuses to reach a terminal state before attempting transfer.
	// Terminal statuses: completed, classification_failed, extraction_failed
	for _, doc := range docs {
		switch doc.Status {
		case entity.DocStatusPending, entity.DocStatusClassifying, entity.DocStatusClassified, entity.DocStatusExtracting, "processing":
			return // still in progress
		}
	}

	// Все документы готовы — пробуем перевести
	err = c.appUseCase.TransferToOperator(ctx, applicantID)
	if err != nil {
		// Не хватает обязательных документов — уведомляем менеджера
		log.Printf("[AUTO-TRANSFER] Applicant %d: transfer blocked: %v", applicantID, err)
		c.hub.BroadcastStatus(applicantID, map[string]interface{}{
			"event":   "analysis_done",
			"missing": err.Error(),
		})
		return
	}

	log.Printf("[AUTO-TRANSFER] Applicant %d transferred to verifying", applicantID)
	c.hub.BroadcastStatus(applicantID, map[string]interface{}{
		"event":  "status_changed",
		"status": "verifying",
	})
}

// failTask updates the queue task status, optionally persists docStatus to the document row,
// broadcasts the failure over WebSocket, and nacks the AMQP message.
// Pass docID=0 to skip the document status update (e.g. when the doc was not found).
func (c *Consumer) failTask(applicantID int64, taskID string, category string, docID int64, docStatus string, d amqp.Delivery, err error, requeue bool) {
	log.Printf(" [x] Failed Task %s (requeue=%v): %v", taskID, requeue, err)
	errMsg := err.Error()
	_ = c.queueRepo.UpdateStatus(context.Background(), taskID, "failed", &errMsg)

	wsBroadcastStatus := docStatus
	if docID > 0 {
		if requeue {
			// Task will be retried — reset doc to "pending" so the UI shows it as
			// "waiting" rather than stuck in an intermediate state (classifying/extracting).
			_ = c.docRepo.UpdateDocumentStatus(context.Background(), docID, entity.DocStatusPending)
			wsBroadcastStatus = entity.DocStatusPending
		} else {
			// Final attempt — persist the terminal error status.
			if docStatus != "" {
				_ = c.docRepo.UpdateDocumentStatus(context.Background(), docID, docStatus)
			}
		}
	}

	c.hub.BroadcastStatus(applicantID, map[string]interface{}{
		"task_id":  taskID,
		"category": category,
		"status":   wsBroadcastStatus,
		"error":    errMsg,
	})
	d.Nack(false, requeue)
}

func getKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
