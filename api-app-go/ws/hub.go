package ws

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// writeTimeout bounds a single broadcast write. Without it, one dead-but-not-
// closed client could block WriteMessage indefinitely and stall the consumer
// goroutine (and, while the lock was held, all register/unregister calls).
const writeTimeout = 5 * time.Second

var upgrader = websocket.Upgrader{
	// Allow all origins — matches Java's open CORS policy
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub maintains the set of active WebSocket connections and broadcasts
// messages to all of them. Mirrors MessageWebSocket.java.
type Hub struct {
	mu    sync.RWMutex
	conns map[*websocket.Conn]struct{}
}

// NewHub creates an initialised Hub.
func NewHub() *Hub {
	return &Hub{conns: make(map[*websocket.Conn]struct{})}
}

func (h *Hub) register(conn *websocket.Conn) {
	h.mu.Lock()
	h.conns[conn] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) unregister(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.conns, conn)
	h.mu.Unlock()
}

// Broadcast sends msg as a text frame to every connected client, each write
// bounded by writeTimeout. A failed/timed-out write closes that connection so
// its read pump wakes and unregisters it; delivery to the other clients is
// unaffected.
func (h *Hub) Broadcast(msg string) {
	// Snapshot the connections so a slow client can't block register/unregister
	// (which need the write lock) while we write. Broadcast is only ever called
	// from the single consumer goroutine, so writes are never concurrent.
	h.mu.RLock()
	conns := make([]*websocket.Conn, 0, len(h.conns))
	for conn := range h.conns {
		conns = append(conns, conn)
	}
	h.mu.RUnlock()

	for _, conn := range conns {
		_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			log.Printf("ws: broadcast write error: %v — closing", err)
			conn.Close() // wakes the read pump; it unregisters the connection
		}
	}
}

// ServeWS upgrades the HTTP connection to WebSocket, registers it in the hub,
// then runs a read pump so disconnections are detected and the connection is
// cleaned up.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws: upgrade error: %v", err)
		return
	}

	h.register(conn)
	log.Println("ws: client connected")

	// Read pump — keeps the connection alive and handles client disconnects
	defer func() {
		h.unregister(conn)
		conn.Close()
		log.Println("ws: client disconnected")
	}()

	for {
		// We don't expect inbound messages from the browser, but we must
		// read to receive control frames (ping/pong/close).
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
