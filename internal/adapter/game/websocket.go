package game

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"rlapi2mqtt/internal/port"
)

type envelope struct {
	Event string          `json:"Event"`
	Data  json.RawMessage `json:"Data"`
}

type RLClient struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	done   chan struct{}
	closed bool
}

func NewRLClient() *RLClient {
	return &RLClient{}
}

// NormalizeAddress turns the stored address into a ws:// URL.
// It accepts both a bare "host:port" (most common, no scheme) and a
// complete "ws://" or "wss://" URL, defaulting to the WebSocket endpoint.
func NormalizeAddress(address string) string {
	addr := strings.TrimSpace(address)
	if addr == "" {
		return "ws://127.0.0.1:49124"
	}
	if strings.HasPrefix(addr, "ws://") || strings.HasPrefix(addr, "wss://") {
		return addr
	}
	return "ws://" + addr
}

func (r *RLClient) Connect(address string, onEvent port.GameEventCallback) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn != nil {
		return fmt.Errorf("already connected")
	}

	u := NormalizeAddress(address)
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.Dial(u, nil)
	if err != nil {
		return fmt.Errorf("WebSocket dial %s failed: %w", u, err)
	}

	r.conn = conn
	r.done = make(chan struct{})
	r.closed = false

	go r.readLoop(onEvent)

	return nil
}

func (r *RLClient) readLoop(onEvent port.GameEventCallback) {
	defer func() {
		r.mu.Lock()
		if r.conn != nil {
			r.conn.Close()
			r.conn = nil
		}
		r.closed = true
		r.mu.Unlock()
	}()

	for {
		select {
		case <-r.done:
			return
		default:
		}

		// Each WebSocket message is one envelope frame (the new Stats API
		// WebSocket transport). ReadMessage handles the frame protocol.
		_, msg, err := r.conn.ReadMessage()
		if err != nil {
			return // connection dropped -> companion reconnect loop handles it
		}

		var env envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			continue // malformed frame; skip
		}

		raw, _ := json.Marshal(env)

		onEvent(port.GameEvent{
			Event: env.Event,
			Data:  raw,
		})
	}
}

func (r *RLClient) Disconnect() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn != nil && !r.closed {
		close(r.done)
		r.conn.Close()
		r.conn = nil
		r.closed = true
	}
}

func (r *RLClient) IsConnected() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.conn != nil && !r.closed
}
