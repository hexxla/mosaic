package mcpsrv

import (
	"context"
	"log/slog"

	ratchetdomain "github.com/hexxla/mcp-ratchet/pkg/ratchet/domain"
	ratchetprimary "github.com/hexxla/mcp-ratchet/pkg/ratchet/ports/primary"
	ratchetsecondary "github.com/hexxla/mcp-ratchet/pkg/ratchet/ports/secondary"
)

// RatchetWrapper holds ratchet service dependencies for tool wrapping.
type RatchetWrapper struct {
	ratchetSvc   ratchetprimary.RatchetService
	sessionStore ratchetsecondary.SessionStore
	log          *slog.Logger
}

// NewRatchetWrapper creates a new ratchet wrapper.
func NewRatchetWrapper(ratchetSvc ratchetprimary.RatchetService, sessionStore ratchetsecondary.SessionStore, log *slog.Logger) *RatchetWrapper {
	return &RatchetWrapper{
		ratchetSvc:   ratchetSvc,
		sessionStore: sessionStore,
		log:          log,
	}
}

// DeriveSessionID extracts a session ID from the MCP context.
// It derives the session ID from the actual MCP session context.
// For now, use a fixed session ID to enable session sharing across calls.
// TODO: Derive from actual MCP session context when available.
func (rw *RatchetWrapper) DeriveSessionID(ctx context.Context) ratchetdomain.SessionID {
	// Use a fixed session ID to enable session sharing across tool calls
	// This allows stateless clients to share session state
	return ratchetdomain.SessionID("mosaic-mcp-session")
}
