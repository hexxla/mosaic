package mcpsrv

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	ratchetdomain "github.com/hexxla/mcp-ratchet/pkg/ratchet/domain"
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
	server := httptest.NewServer(observer)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?session_id=session-a"
	header := http.Header{"Authorization": {"Bearer " + token}}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("dial observer: %v", err)
	}
	defer func() { _ = conn.Close() }()

	wantTime := time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC)
	if err := observer.Store(context.Background(), &ratchetdomain.Event{
		ID:        "ignored-session",
		Type:      ratchetdomain.EventTypeToolCallAttempt,
		SessionID: "session-b",
		Timestamp: wantTime,
	}); err != nil {
		t.Fatalf("store ignored event: %v", err)
	}
	if err := observer.Store(context.Background(), &ratchetdomain.Event{
		ID:        "event-1",
		Type:      ratchetdomain.EventTypeTokenCreated,
		SessionID: "session-a",
		ToolName:  "mosaic_hexxla_query_cells",
		Token:     ratchetdomain.TokenValue(strings.Repeat("secret", 8)),
		Timestamp: wantTime,
		Metadata:  map[string]any{"private": "metadata"},
	}); err != nil {
		t.Fatalf("store event: %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read event: %v", err)
	}
	if strings.Contains(string(payload), "secret") || strings.Contains(string(payload), "metadata") {
		t.Fatalf("credential-bearing fields leaked: %s", payload)
	}
	var got RatchetEvent
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if got.ID != "event-1" || got.SessionID != "session-a" || got.ToolName != "mosaic_hexxla_query_cells" || !got.Timestamp.Equal(wantTime) {
		t.Fatalf("unexpected event: %+v", got)
	}

	events, err := observer.GetEvents(context.Background(), "session-a", nil)
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("live observer retained %d events", len(events))
	}
}
