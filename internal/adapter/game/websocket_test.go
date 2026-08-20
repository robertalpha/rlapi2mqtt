package game

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"rlapi2mqtt/internal/port"
)

func serveWS(t *testing.T, frames ...string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for _, f := range frames {
			if err := c.WriteMessage(websocket.TextMessage, []byte(f)); err != nil {
				return
			}
		}
		if len(frames) == 0 {
			time.Sleep(300 * time.Millisecond) // keep the connection alive briefly
		}
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)
	return strings.TrimPrefix(srv.URL, "http://")
}

func TestRLClient_WebSocketDecodesEnvelope(t *testing.T) {
	const frame = `{"Event":"UpdateState","Data":{"Players":[],"Game":{"TimeSeconds":5}}}`
	addr := serveWS(t, frame)

	c := NewRLClient()
	got := make(chan port.GameEvent, 1)
	if err := c.Connect(addr, func(e port.GameEvent) { got <- e }); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer c.Disconnect()

	select {
	case ev := <-got:
		if ev.Event != "UpdateState" {
			t.Fatalf("event = %q, want UpdateState", ev.Event)
		}
		if !strings.Contains(string(ev.Data), "UpdateState") ||
			!strings.Contains(string(ev.Data), "TimeSeconds") {
			t.Fatalf("data = %s, want envelope to be forwarded verbatim", ev.Data)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestRLClient_ConnectPrefersWsAddr(t *testing.T) {
	addr := serveWS(t)
	c := NewRLClient()
	defer c.Disconnect()

	// The test server returns an address that, prefixed with ws://, is a full
	// valid WebSocket URL — proving the ws:// scheme works end to end.
	if err := c.Connect("ws://"+addr, func(e port.GameEvent) {}); err != nil {
		t.Fatalf("connect with ws:// prefix: %v", err)
	}
}

func TestNormalizeAddress(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "ws://127.0.0.1:49124"},
		{"127.0.0.1:49124", "ws://127.0.0.1:49124"},
		{" localhost:49124 ", "ws://localhost:49124"},
		{"ws://127.0.0.1:49124", "ws://127.0.0.1:49124"},
		{"ws://127.0.0.1:49124/", "ws://127.0.0.1:49124/"},
		{"wss://example.com", "wss://example.com"},
	}
	for _, c := range cases {
		if got := NormalizeAddress(c.in); got != c.want {
			t.Errorf("NormalizeAddress(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
