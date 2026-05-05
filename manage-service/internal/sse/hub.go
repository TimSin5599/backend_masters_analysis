package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type client struct {
	ch chan []byte
}

// Hub manages active SSE connections keyed by applicant ID.
type Hub struct {
	mu      sync.RWMutex
	clients map[int64][]*client
}

func NewHub() *Hub {
	return &Hub{clients: make(map[int64][]*client)}
}

func (h *Hub) subscribe(applicantID int64) *client {
	cl := &client{ch: make(chan []byte, 16)}
	h.mu.Lock()
	h.clients[applicantID] = append(h.clients[applicantID], cl)
	h.mu.Unlock()
	log.Printf("SSE: client subscribed for applicant %d", applicantID)
	return cl
}

func (h *Hub) unsubscribe(applicantID int64, cl *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	conns := h.clients[applicantID]
	for i, c := range conns {
		if c == cl {
			h.clients[applicantID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
	if len(h.clients[applicantID]) == 0 {
		delete(h.clients, applicantID)
	}
	log.Printf("SSE: client unsubscribed for applicant %d", applicantID)
}

// BroadcastStatus sends a JSON payload to all SSE subscribers for an applicantID.
func (h *Hub) BroadcastStatus(applicantID int64, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("SSE: failed to marshal payload: %v", err)
		return
	}
	h.mu.RLock()
	clients := make([]*client, len(h.clients[applicantID]))
	copy(clients, h.clients[applicantID])
	h.mu.RUnlock()

	for _, cl := range clients {
		select {
		case cl.ch <- data:
		default:
			log.Printf("SSE: slow client dropped a message for applicant %d", applicantID)
		}
	}
}

// HandleSSE is the Gin handler for GET /v1/applicants/:id/status/stream.
func (h *Hub) HandleSSE(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	cl := h.subscribe(applicantID)
	defer h.unsubscribe(applicantID, cl)

	// Disable the server-level WriteTimeout for this streaming connection.
	// net/http sets a write deadline after reading the request; without this
	// reset the connection is killed after WriteTimeout (default 5s), which
	// causes all messages that arrive after the timeout to be silently dropped.
	rc := http.NewResponseController(c.Writer)
	_ = rc.SetWriteDeadline(time.Time{})

	// Flush headers before entering the loop so the client gets the 200 immediately.
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	ctx := c.Request.Context()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			fmt.Fprint(c.Writer, ": heartbeat\n\n")
			c.Writer.Flush()
		case data := <-cl.ch:
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
			c.Writer.Flush()
		}
	}
}
