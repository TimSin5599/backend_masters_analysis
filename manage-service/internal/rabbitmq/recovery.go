package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"time"

	"manage-service/internal/domain/entity"
	"manage-service/internal/usecase"
)

// recoverStuckTasks re-enqueues tasks stuck in "processing" or "pending" for
// longer than threshold. Handles scenarios where the service crashed mid-processing
// and RabbitMQ lost the in-flight message (unacked messages are normally redelivered
// automatically, but this covers edge cases like broker restarts).
//
// It also resets the associated document status back to "pending" so the UI
// does not show a stale intermediate state (classifying/extracting).
func recoverStuckTasks(ctx context.Context, queueRepo usecase.DocumentQueueRepo, docRepo usecase.DocumentRepo, producer usecase.DocumentQueueProducer, threshold time.Duration) {
	tasks, err := queueRepo.GetStuckTasks(ctx, threshold)
	if err != nil {
		log.Printf("[RECOVERY] Failed to query stuck tasks: %v", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	log.Printf("[RECOVERY] Found %d stuck task(s) (threshold: %s), re-enqueuing...", len(tasks), threshold)

	for _, task := range tasks {
		age := time.Since(task.UpdatedAt).Round(time.Second)
		log.Printf("[RECOVERY] Task %s | status=%s | age=%s | file=%s", task.ID, task.Status, age, task.FilePath)

		// Reset the associated document to "pending" so the frontend shows it
		// as waiting rather than stuck in classifying/extracting.
		if resetErr := docRepo.UpdateDocumentStatusByPath(ctx, task.FilePath, entity.DocStatusPending); resetErr != nil {
			log.Printf("[RECOVERY] Warning: failed to reset document status for path %s: %v", task.FilePath, resetErr)
		}

		if err := queueRepo.UpdateStatus(ctx, task.ID, "pending", nil); err != nil {
			log.Printf("[RECOVERY] Failed to reset task %s status: %v", task.ID, err)
			continue
		}

		task.Status = "pending"
		if err := producer.PublishTask(task); err != nil {
			errMsg := fmt.Sprintf("recovery publish failed: %v", err)
			log.Printf("[RECOVERY] ❌ Failed to re-publish task %s: %v", task.ID, err)
			_ = queueRepo.UpdateStatus(ctx, task.ID, "failed", &errMsg)
		} else {
			log.Printf("[RECOVERY] ✅ Task %s re-enqueued", task.ID)
		}
	}
}

// StartPeriodicRecovery runs recoverStuckTasks on startup (after an initial delay
// to let the consumer pick up existing RabbitMQ messages first) and then every
// interval until ctx is cancelled.
//
// threshold should be longer than the consumer's max processing timeout (12 min)
// to avoid touching tasks that are still actively being processed.
func StartPeriodicRecovery(ctx context.Context, queueRepo usecase.DocumentQueueRepo, docRepo usecase.DocumentRepo, producer usecase.DocumentQueueProducer, interval, threshold time.Duration) {
	// Wait after startup so the consumer can pick up any messages already in
	// RabbitMQ before we decide they are "stuck".
	select {
	case <-ctx.Done():
		return
	case <-time.After(30 * time.Second):
	}

	recoverStuckTasks(ctx, queueRepo, docRepo, producer, threshold)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			recoverStuckTasks(ctx, queueRepo, docRepo, producer, threshold)
		}
	}
}
