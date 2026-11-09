package mcpsrv

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hexxla/mcp-ratchet/pkg/ratchet"
)

func TestRatchetObserverRequiresAuthenticationAndRejectsOrigins(t *testing.T) {
	t.Parallel()

	token := strings.Repeat("a", 32)
	server := httptest.NewServer(NewRatchetObserver([]byte(token)))
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	tests := []struct {
		name       string
		header     http.Header
		wantStatus int
	}{
		{name: "missing bearer", header: http.Header{}, wantStatus: http.StatusUnauthorized},
		{name: "wrong bearer", header: http.Header{"Authorization": {"Bearer wrong"}}, wantStatus: http.StatusUnauthorized},
		{name: "browser origin", header: http.Header{"Authorization": {"Bearer " + token}, "Origin": {"http://127.0.0.1"}}, wantStatus: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, response, err := websocket.DefaultDialer.Dial(wsURL, tt.header)
			if err == nil {
				t.Fatal("expected upgrade failure")
			}
			if response == nil || response.StatusCode != tt.wantStatus {
				t.Fatalf("status = %v, want %d", response, tt.wantStatus)
			}
		})
	}
}

func TestRatchetObserverStreamsRedactedSessionEvents(t *testing.T) {
	t.Parallel()

	token := strings.Repeat("b", 32)
	observer := NewRatchetObserver([]byte(token))
	if observer.sessionLabel("session-a") == NewRatchetObserver([]byte(token)).sessionLabel("session-a") {
		t.Fatal("observer instances reused a session label")
	}
	server := httptest.NewServer(observer)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?session_id=" + observer.sessionLabel("session-a")
	header := http.Header{"Authorization": {"Bearer " + token}}
	rawURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?session_id=session-a"
	if raw, response, err := websocket.DefaultDialer.Dial(rawURL, header); err == nil {
		_ = raw.Close()
		t.Fatal("raw session ID accepted as an observer filter")
	} else if response == nil || response.StatusCode != http.StatusBadRequest {
		t.Fatalf("raw session filter response = %v", response)
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("dial observer: %v", err)
	}
	defer func() { _ = conn.Close() }()

	wantTime := time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC)
	observer.Observe(ratchet.Event{
		ID:        "ignored-session",
		Type:      ratchet.EventTypeToolCallAttempt,
		SessionID: "session-b",
		Timestamp: wantTime,
	})
	observer.Observe(ratchet.Event{
		ID:         "event-1",
		Type:       ratchet.EventTypeCompletionRecorded,
		SessionID:  "session-a",
		ToolName:   "mosaic_hexxla_search_cells",
		Timestamp:  wantTime,
		Error:      "secret metadata",
		ConsumedBy: "secret",
	})

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read event: %v", err)
	}
	if strings.Contains(string(payload), "secret") || strings.Contains(string(payload), "metadata") || strings.Contains(string(payload), "session-a") {
		t.Fatalf("credential-bearing fields leaked: %s", payload)
	}
	var got RatchetEvent
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if got.ID != "event-1" || got.SessionID != observer.sessionLabel("session-a") || got.ToolName != "mosaic_hexxla_search_cells" || !got.Timestamp.Equal(wantTime) {
		t.Fatalf("unexpected event: %+v", got)
	}

}
