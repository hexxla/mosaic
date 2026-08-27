package mcpsrv

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	ratchetdomain "github.com/hexxla/mcp-ratchet/pkg/ratchet/domain"
	ratchetsecondary "github.com/hexxla/mcp-ratchet/pkg/ratchet/ports/secondary"
)

const (
	observerEventBuffer = 64
	observerMaxClients  = 16
	observerWriteWait   = 10 * time.Second
	observerPongWait    = 60 * time.Second
	observerPingPeriod  = 45 * time.Second
	observerMaxMessage  = 1024
)

// RatchetEvent is the credential-free event shape sent to authenticated live
// observers. Ratchet capability tokens and event metadata are intentionally
// excluded from the network representation.
type RatchetEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	SessionID string    `json:"session_id"`
	ToolName  string    `json:"tool_name,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type ratchetSubscriber struct {
	sessionID ratchetdomain.SessionID
	events    chan RatchetEvent
}

// RatchetObserver is a bounded, live-only Ratchet EventStore and authenticated
// WebSocket handler. It retains no historical events.
type RatchetObserver struct {
	mu          sync.RWMutex
	nextID      uint64
	subscribers map[uint64]ratchetSubscriber
	token       []byte
	upgrader    websocket.Upgrader
}

// NewRatchetObserver creates an authenticated live event stream. The caller
// must provide a validated bearer token of at least 32 bytes.
func NewRatchetObserver(token []byte) *RatchetObserver {
	return &RatchetObserver{
		subscribers: make(map[uint64]ratchetSubscriber),
		token:       append([]byte(nil), token...),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return r.Header.Get("Origin") == ""
			},
		},
	}
}

// Store implements Ratchet's EventStore as a live, non-retaining sink.
func (o *RatchetObserver) Store(_ context.Context, event *ratchetdomain.Event) error {
	if event == nil {
		return nil
	}
	o.broadcast(event)
	return nil
}

// GetEvents returns no events because the observer intentionally retains no history.
func (*RatchetObserver) GetEvents(context.Context, ratchetdomain.SessionID, *ratchetsecondary.EventFilter) ([]*ratchetdomain.Event, error) {
	return []*ratchetdomain.Event{}, nil
}

// GetStats returns zero-valued live-only statistics because no history is retained.
func (*RatchetObserver) GetStats(context.Context) (*ratchetdomain.EventStats, error) {
	return &ratchetdomain.EventStats{
		EventsByType: make(map[ratchetdomain.EventType]int),
		EventsByTool: make(map[ratchetdomain.ToolName]int),
	}, nil
}

// ServeHTTP upgrades an authenticated, non-browser request to a WebSocket. An
// optional session_id query limits delivery to one MCP session; omitting it
// subscribes the authenticated operator to all live sessions.
func (o *RatchetObserver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !o.authorized(r) {
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Header.Get("Origin") != "" {
		http.Error(w, "browser-origin WebSocket requests are not allowed", http.StatusForbidden)
		return
	}

	sessionID, ok := observerSessionID(r)
	if !ok {
		http.Error(w, "invalid session_id", http.StatusBadRequest)
		return
	}
	subscriberID, subscriber, ok := o.subscribe(sessionID)
	if !ok {
		http.Error(w, "observer connection limit reached", http.StatusServiceUnavailable)
		return
	}

	conn, err := o.upgrader.Upgrade(w, r, nil)
	if err != nil {
		o.unsubscribe(subscriberID)
		return
	}
	defer func() {
		o.unsubscribe(subscriberID)
		_ = conn.Close()
	}()

	conn.SetReadLimit(observerMaxMessage)
	_ = conn.SetReadDeadline(time.Now().Add(observerPongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(observerPongWait))
	})

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ping := time.NewTicker(observerPingPeriod)
	defer ping.Stop()
	for {
		select {
		case event := <-subscriber.events:
			if err := conn.SetWriteDeadline(time.Now().Add(observerWriteWait)); err != nil {
				return
			}
			if err := conn.WriteJSON(event); err != nil {
				return
			}
		case <-ping.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(observerWriteWait)); err != nil {
				return
			}
		case <-readDone:
			return
		case <-r.Context().Done():
			return
		}
	}
}

func (o *RatchetObserver) authorized(r *http.Request) bool {
	values := r.Header.Values("Authorization")
	if len(values) != 1 {
		return false
	}
	const prefix = "Bearer "
	header := values[0]
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(header[len(prefix):]), o.token) == 1
}

func observerSessionID(r *http.Request) (ratchetdomain.SessionID, bool) {
	values, present := r.URL.Query()["session_id"]
	if !present {
		return "", true
	}
	if len(values) != 1 || len(values[0]) > 256 {
		return "", false
	}
	sessionID := ratchetdomain.SessionID(values[0])
	if err := sessionID.Validate(); err != nil {
		return "", false
	}
	return sessionID, true
}

func (o *RatchetObserver) subscribe(sessionID ratchetdomain.SessionID) (uint64, ratchetSubscriber, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.subscribers) >= observerMaxClients {
		return 0, ratchetSubscriber{}, false
	}
	o.nextID++
	subscriber := ratchetSubscriber{
		sessionID: sessionID,
		events:    make(chan RatchetEvent, observerEventBuffer),
	}
	o.subscribers[o.nextID] = subscriber
	return o.nextID, subscriber, true
}

func (o *RatchetObserver) unsubscribe(id uint64) {
	o.mu.Lock()
	delete(o.subscribers, id)
	o.mu.Unlock()
}

func (o *RatchetObserver) broadcast(event *ratchetdomain.Event) {
	redacted := RatchetEvent{
		ID:        string(event.ID),
		Type:      string(event.Type),
		SessionID: string(event.SessionID),
		ToolName:  string(event.ToolName),
		Timestamp: event.Timestamp,
	}
	o.mu.RLock()
	defer o.mu.RUnlock()
	for _, subscriber := range o.subscribers {
		if subscriber.sessionID != "" && subscriber.sessionID != event.SessionID {
			continue
		}
		select {
		case subscriber.events <- redacted:
		default:
			// Observability is best-effort and must never stall tool enforcement.
		}
	}
}
