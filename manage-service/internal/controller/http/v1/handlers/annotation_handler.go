package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"manage-service/internal/usecase"
)

type annotationJob struct {
	Status     string // "generating" | "done" | "error"
	Annotation string
	Err        string
}

// annotationCache holds in-progress and completed annotation jobs keyed by applicant ID.
var annotationCache sync.Map

type AnnotationHandler struct {
	uc        usecase.Applicant
	extractor usecase.ExtractionClient
}

func NewAnnotationHandler(uc usecase.Applicant, extractor usecase.ExtractionClient) *AnnotationHandler {
	return &AnnotationHandler{uc: uc, extractor: extractor}
}

// StreamAnnotation handles GET /v1/applicants/:id/annotation/stream as a Server-Sent Events endpoint.
// It starts AI generation (if not already running) and streams the result back to the client.
// Heartbeat comments are sent every 15 s to keep proxies from closing the connection.
// Pass ?regenerate=true to discard any cached result and force a new generation.
func (h *AnnotationHandler) StreamAnnotation(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Disable the server-level WriteTimeout so the long-running SSE stream is
	// not killed after the default 5-second write deadline.
	rc := http.NewResponseController(c.Writer)
	_ = rc.SetWriteDeadline(time.Time{})

	idStr := c.Param("id")
	applicantID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		sseWrite(c, "error", map[string]string{"error": "invalid applicant id"})
		return
	}

	// ?regenerate=true clears any cached result (done or error) so a fresh job starts.
	if c.Query("regenerate") == "true" {
		annotationCache.Delete(applicantID)
	}

	// Return cached result immediately if already done.
	if existing, ok := annotationCache.Load(applicantID); ok {
		job := existing.(*annotationJob)
		if job.Status == "done" {
			sseWrite(c, "done", map[string]string{"annotation": job.Annotation})
			return
		}
		// Cached error is not returned — fall through to start a new job.
		// (Error state is cleared by the Delete above when regenerate=true;
		//  without regenerate, a fresh connection still retries rather than
		//  replaying the old error.)
		if job.Status == "error" {
			annotationCache.Delete(applicantID)
		}
	}

	// Start generation if not already running.
	if existing, ok := annotationCache.Load(applicantID); !ok || existing.(*annotationJob).Status != "generating" {
		job := &annotationJob{Status: "generating"}
		annotationCache.Store(applicantID, job)
		go h.runAnnotationJob(applicantID, job)
	}

	sseWrite(c, "status", map[string]string{"status": "generating"})
	c.Writer.Flush()

	ctx := c.Request.Context()
	heartbeat := time.NewTicker(15 * time.Second)
	check := time.NewTicker(500 * time.Millisecond)
	defer heartbeat.Stop()
	defer check.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-heartbeat.C:
			// SSE comment — keeps the TCP connection alive through proxies.
			fmt.Fprint(c.Writer, ": heartbeat\n\n")
			c.Writer.Flush()

		case <-check.C:
			existing, ok := annotationCache.Load(applicantID)
			if !ok {
				continue
			}
			job := existing.(*annotationJob)
			switch job.Status {
			case "done":
				sseWrite(c, "done", map[string]string{"annotation": job.Annotation})
				c.Writer.Flush()
				return
			case "error":
				// "annotation_error" avoids collision with the browser's built-in
				// EventSource connection-level "error" event.
				sseWrite(c, "annotation_error", map[string]string{"error": job.Err})
				c.Writer.Flush()
				return
			}
		}
	}
}

func sseWrite(c *gin.Context, event string, payload interface{}) {
	data, _ := json.Marshal(payload)
	fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data)
	c.Writer.Flush()
}

func (h *AnnotationHandler) runAnnotationJob(applicantID int64, job *annotationJob) {
	categories := []string{
		"personal_data", "diploma", "transcript",
		"prof_development", "second_diploma", "certification",
		"achievement", "motivation", "recommendation", "language",
	}

	applicantData := make(map[string]interface{})
	for _, cat := range categories {
		catData, err := h.uc.GetApplicantData(context.Background(), applicantID, cat)
		if err != nil || catData == nil {
			continue
		}
		b, err := json.Marshal(catData)
		if err != nil {
			continue
		}
		raw := string(b)
		if raw != "null" && raw != "[]" && raw != "{}" {
			applicantData[cat] = raw
		}
	}

	annotation, err := h.extractor.GenerateAnnotation(context.Background(), applicantData)
	if err != nil {
		job.Status = "error"
		job.Err = err.Error()
	} else {
		job.Status = "done"
		job.Annotation = annotation
	}
	annotationCache.Store(applicantID, job)
}
