package mcpsrv

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hexxla/mcp-ratchet/pkg/ratchet"
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
	sessionID string
	events    chan RatchetEvent
}

// RatchetObserver is a bounded, live-only Ratchet observer and authenticated
// WebSocket handler. It retains no historical events.
type RatchetObserver struct {
	mu          sync.RWMutex
	nextID      uint64
	subscribers map[uint64]ratchetSubscriber
	token       []byte
	labelKey    [32]byte
	upgrader    websocket.Upgrader
}

// NewRatchetObserver creates an authenticated live event stream. The caller
// must provide a validated bearer token of at least 32 bytes.
func NewRatchetObserver(token []byte) *RatchetObserver {
	observer := &RatchetObserver{
		subscribers: make(map[uint64]ratchetSubscriber),
		token:       bytes.Clone(token),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return r.Header.Get("Origin") == ""
			},
		},
	}
	rand.Read(observer.labelKey[:])
	return observer
}

// Observe receives immutable core events without retaining history.
func (o *RatchetObserver) Observe(event ratchet.Event) { o.broadcast(event) }

// ServeHTTP upgrades an authenticated, non-browser request to a WebSocket. An
// optional session_id query filters by an observer correlation ID; omitting it
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

func observerSessionID(r *http.Request) (string, bool) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return "", false
	}
	if len(query) == 0 {
		return "", true
	}
	values := query["session_id"]
	if len(query) != 1 {
		return "", false
	}
	if len(values) != 1 || len(values[0]) != 32 {
		return "", false
	}
	if _, err := hex.DecodeString(values[0]); err != nil {
		return "", false
	}
	return strings.ToLower(values[0]), true
}

func (o *RatchetObserver) subscribe(sessionID string) (uint64, ratchetSubscriber, bool) {
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

func (o *RatchetObserver) sessionLabel(id string) string {
	mac := hmac.New(sha256.New, o.labelKey[:])
	_, _ = mac.Write([]byte(id))
	return hex.EncodeToString(mac.Sum(nil)[:16])
}

func (o *RatchetObserver) broadcast(event ratchet.Event) {
	redacted := RatchetEvent{
		ID:        string(event.ID),
		Type:      string(event.Type),
		SessionID: o.sessionLabel(event.SessionID),
		ToolName:  string(event.ToolName),
		Timestamp: event.Timestamp,
	}
	o.mu.RLock()
	defer o.mu.RUnlock()
	for _, subscriber := range o.subscribers {
		if subscriber.sessionID != "" && subscriber.sessionID != redacted.SessionID {
			continue
		}
		select {
		case subscriber.events <- redacted:
		default:
			// Observability is best-effort and must never stall tool enforcement.
		}
	}
}
