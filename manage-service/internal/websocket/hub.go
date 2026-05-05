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

// managedConn wraps a websocket.Conn with a write mutex.
// Gorilla websocket allows one concurrent reader and one concurrent writer —
// without this lock, the heartbeat ping and BroadcastStatus race on writes.
type managedConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
	done chan struct{}
}

func newManagedConn(conn *websocket.Conn) *managedConn {
	return &managedConn{conn: conn, done: make(chan struct{})}
}

func (m *managedConn) writeMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.conn.WriteMessage(messageType, data)
}

func (m *managedConn) writeJSON(v interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.conn.WriteJSON(v)
}

// shutdown closes the connection and signals the heartbeat goroutine to stop.
// Safe to call multiple times.
func (m *managedConn) shutdown() {
	select {
	case <-m.done:
		// already closed
	default:
		close(m.done)
		m.mu.Lock()
		m.conn.Close()
		m.mu.Unlock()
	}
}

// Hub manages active WebSocket connections
type Hub struct {
	connections map[int64][]*managedConn
	mu          sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		connections: make(map[int64][]*managedConn),
	}
}

// HandleWebSocket upgrades the HTTP request to a WebSocket and registers it
func (h *Hub) HandleWebSocket(c *gin.Context, applicantID int64) {
	rawConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	mc := newManagedConn(rawConn)
	h.register(applicantID, mc)

	go func() {
		defer func() {
			h.unregister(applicantID, mc)
			mc.shutdown()
		}()

		// Heartbeat — stops immediately when shutdown() closes mc.done
		go func() {
			ticker := time.NewTicker(50 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-mc.done:
					return
				case <-ticker.C:
					if err := mc.writeMessage(websocket.PingMessage, nil); err != nil {
						return
					}
				}
			}
		}()

		rawConn.SetReadDeadline(time.Now().Add(60 * time.Second))
		rawConn.SetPongHandler(func(string) error {
			return rawConn.SetReadDeadline(time.Now().Add(60 * time.Second))
		})

		for {
			_, _, err := rawConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error: %v", err)
				}
				break
			}
		}
	}()
}

func (h *Hub) register(applicantID int64, mc *managedConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections[applicantID] = append(h.connections[applicantID], mc)
	log.Printf("WS: Registered client for applicant %d", applicantID)
}

func (h *Hub) unregister(applicantID int64, mc *managedConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	conns := h.connections[applicantID]
	for i, c := range conns {
		if c == mc {
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

	for _, mc := range conns {
		if err := mc.writeJSON(payload); err != nil {
			log.Printf("WS: Failed to send message to applicant %d: %v", applicantID, err)
			mc.shutdown()
			h.unregister(applicantID, mc)
		}
	}
}
