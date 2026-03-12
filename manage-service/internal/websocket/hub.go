package ws

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity in this project
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Hub manages active WebSocket connections
type Hub struct {
	// connections maps applicant_id -> list of active connections
	connections map[int64][]*websocket.Conn
	mu          sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		connections: make(map[int64][]*websocket.Conn),
	}
}

// HandleWebSocket upgrades the HTTP request to a WebSocket and registers it
func (h *Hub) HandleWebSocket(c *gin.Context, applicantID int64) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	h.register(applicantID, conn)

	// Keep the connection open and listen for pings/close messages
	go func() {
		defer func() {
			h.unregister(applicantID, conn)
			conn.Close()
		}()

		// Heartbeat
		go func() {
			ticker := time.NewTicker(50 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					h.mu.RLock()
					// basic sanity check if still there
					h.mu.RUnlock()
					if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						return
					}
				}
			}
		}()

		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error: %v", err)
				}
				break
			}
		}
	}()
}

func (h *Hub) register(applicantID int64, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections[applicantID] = append(h.connections[applicantID], conn)
	log.Printf("WS: Registered client for applicant %d", applicantID)
}

func (h *Hub) unregister(applicantID int64, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	conns := h.connections[applicantID]
	for i, c := range conns {
		if c == conn {
			h.connections[applicantID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
	if len(h.connections[applicantID]) == 0 {
		delete(h.connections, applicantID)
	}
	log.Printf("WS: Unregistered client for applicant %d", applicantID)
}

// BroadcastStatus sends a JSON payload to all connected clients for an applicantID
func (h *Hub) BroadcastStatus(applicantID int64, payload interface{}) {
	h.mu.RLock()
	conns := h.connections[applicantID]
	h.mu.RUnlock()

	for _, conn := range conns {
		err := conn.WriteJSON(payload)
		if err != nil {
			log.Printf("WS: Failed to send message to applicant %d: %v", applicantID, err)
			conn.Close()
			h.unregister(applicantID, conn)
		}
	}
}
