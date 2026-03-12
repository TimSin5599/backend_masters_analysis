package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"manage-service/internal/entity"
	"manage-service/internal/usecase"
	ws "manage-service/internal/websocket"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	conn       *amqp.Connection
	ch         *amqp.Channel
	repo       usecase.ApplicantRepo
	queueRepo  usecase.DocumentQueueRepo
	useCase    *usecase.ApplicantUseCase
	s3         usecase.S3Provider
	extractor  usecase.ExtractionClient
	hub        *ws.Hub
}

func NewConsumer(url string, repo usecase.ApplicantRepo, queueRepo usecase.DocumentQueueRepo, uc *usecase.ApplicantUseCase, s3 usecase.S3Provider, extractor usecase.ExtractionClient, hub *ws.Hub) (*Consumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Declare queue to ensure it exists
	args := amqp.Table{
		"x-max-priority": int32(10),
	}
	_, err = ch.QueueDeclare(
		QueueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		args,  // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare a queue: %w", err)
	}

	// Fetch 1 message at a time
	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return &Consumer{
		conn:      conn,
		ch:        ch,
		repo:      repo,
		queueRepo: queueRepo,
		useCase:   uc,
		s3:        s3,
		extractor: extractor,
		hub:       hub,
	}, nil
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
	msgs, err := c.ch.Consume(
		QueueName, // queue
		"",        // consumer tag
		false,     // auto-ack (IMPORTANT: false means we manually ack after processing)
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %w", err)
	}

	log.Println(" [*] RabbitMQ Consumer started. Waiting for messages.")

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println(" [*] Shutting down RabbitMQ Consumer.")
				return
			case d, ok := <-msgs:
				if !ok {
					log.Println(" [!] RabbitMQ channel closed.")
					return
				}
				c.processMessage(d)
			}
		}
	}()

	return nil
}

func (c *Consumer) processMessage(d amqp.Delivery) {
	var task entity.DocumentQueueTask
	if err := json.Unmarshal(d.Body, &task); err != nil {
		log.Printf(" [!] Failed to unmarshal message body: %v", err)
		d.Nack(false, false) // Reject and do not requeue
		return
	}

	log.Printf(" [->] Processing Document %s (Priority: %d, Category: %s)", task.ID, task.Priority, task.DocumentCategory)

	backgroundCtx := context.Background()

	// Update Status to Processing
	_ = c.queueRepo.UpdateStatus(backgroundCtx, task.ID, "processing", nil)
	c.hub.BroadcastStatus(task.ApplicantID, map[string]interface{}{
		"task_id":  task.ID,
		"category": task.DocumentCategory,
		"status":   "processing",
		"progress": 25,
	})

	// Fetch the actual document from DB to pass to the extractor
	docs, err := c.repo.GetDocuments(backgroundCtx, task.ApplicantID)
	if err != nil {
		c.failTask(task.ApplicantID, task.ID, d, fmt.Errorf("failed to get applicant documents: %w", err))
		return
	}

	// Find the matching document (we'll just use the latest matching category for this applicant)
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
		c.failTask(task.ApplicantID, task.ID, d, fmt.Errorf("document metadata not found in db for path %s", task.FilePath))
		return
	}

	// Fetch content from S3
	content, err := c.s3.GetFile(backgroundCtx, task.FilePath)
	if err != nil {
		c.failTask(task.ApplicantID, task.ID, d, fmt.Errorf("failed to get file from S3: %w", err))
		return
	}

	// Trigger AI Extraction
	c.hub.BroadcastStatus(task.ApplicantID, map[string]interface{}{
		"task_id":  task.ID,
		"category": task.DocumentCategory,
		"status":   "extracting",
		"progress": 50,
	})
	rawData, err := c.extractor.TriggerExtraction(backgroundCtx, targetDoc, content)
	if err != nil {
		c.failTask(task.ApplicantID, task.ID, d, fmt.Errorf("AI extraction failed: %w", err))
		return
	}

	// Save results via UseCase
	c.hub.BroadcastStatus(task.ApplicantID, map[string]interface{}{
		"task_id":  task.ID,
		"category": task.DocumentCategory,
		"status":   "saving",
		"progress": 75,
	})
	err = c.useCase.ProcessAIResult(backgroundCtx, targetDoc.ApplicantID, targetDoc.ID, task.DocumentCategory, rawData)
	if err != nil {
		c.failTask(task.ApplicantID, task.ID, d, fmt.Errorf("failed to process AI result: %w", err))
		return
	}

	// Success
	_ = c.queueRepo.UpdateStatus(backgroundCtx, task.ID, "completed", nil)
	log.Printf(" [<-] Completed Document %s", task.ID)
	c.hub.BroadcastStatus(task.ApplicantID, map[string]interface{}{
		"task_id":  task.ID,
		"category": task.DocumentCategory,
		"status":   "completed",
		"progress": 100,
	})
	
// Acknowledge the message to remove it from the queue
	d.Ack(false)
}

func (c *Consumer) failTask(applicantID int64, taskID string, d amqp.Delivery, err error) {
	log.Printf(" [x] Failed Task %s: %v", taskID, err)
	errMsg := err.Error()
	_ = c.queueRepo.UpdateStatus(context.Background(), taskID, "failed", &errMsg)
	c.hub.BroadcastStatus(applicantID, map[string]interface{}{
		"task_id":  taskID,
		"status":   "failed",
		"error":    errMsg,
	})
	
	// Reject but don't requeue (to avoid infinite loops on broken documents)
	d.Nack(false, false)
}
