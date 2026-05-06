package game

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"rlapi2mqtt/internal/port"
)

type envelope struct {
	Event string          `json:"Event"`
	Data  json.RawMessage `json:"Data"`
}

type RLClient struct {
	conn   net.Conn
	mu     sync.Mutex
	done   chan struct{}
	closed bool
}

func NewRLClient() *RLClient {
	return &RLClient{}
}

func (r *RLClient) Connect(address string, onEvent port.GameEventCallback) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn != nil {
		return fmt.Errorf("already connected")
	}

	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return fmt.Errorf("TCP dial failed: %w", err)
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

	decoder := json.NewDecoder(r.conn)

	for {
		select {
		case <-r.done:
			return
		default:
		}

		var env envelope
		if err := decoder.Decode(&env); err != nil {
			return
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
