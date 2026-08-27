package mcpsrv

import (
	"context"
	"fmt"
	"hash/maphash"
	"log/slog"
	"slices"
	"strings"
	"sync"

	ratchetdomain "github.com/hexxla/mcp-ratchet/pkg/ratchet/domain"
	ratchetprimary "github.com/hexxla/mcp-ratchet/pkg/ratchet/ports/primary"
	ratchetsecondary "github.com/hexxla/mcp-ratchet/pkg/ratchet/ports/secondary"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const ratchetLockStripes = 64

// RatchetGate enforces configured tool prerequisites for each MCP session.
type RatchetGate struct {
	service    ratchetprimary.RatchetService
	sessions   ratchetsecondary.SessionStore
	configured map[ratchetdomain.ToolName]struct{}
	protected  map[ratchetdomain.ToolName]struct{}
	lockSeed   maphash.Seed
	locks      [ratchetLockStripes]sync.Mutex
	log        *slog.Logger
}

// NewRatchetGate constructs a session-isolated Ratchet middleware.
func NewRatchetGate(
	service ratchetprimary.RatchetService,
	sessions ratchetsecondary.SessionStore,
	rules []ratchetdomain.Rule,
	log *slog.Logger,
) *RatchetGate {
	configured := make(map[ratchetdomain.ToolName]struct{}, len(rules))
	withPrerequisite := make(map[ratchetdomain.ToolName]struct{}, len(rules))
	withFreePass := make(map[ratchetdomain.ToolName]struct{}, len(rules))
	for _, rule := range rules {
		configured[rule.Tool] = struct{}{}
		if rule.Prerequisite == "" {
			withFreePass[rule.Tool] = struct{}{}
		} else {
			withPrerequisite[rule.Tool] = struct{}{}
		}
	}
	protected := make(map[ratchetdomain.ToolName]struct{}, len(withPrerequisite))
	for tool := range withPrerequisite {
		if _, unrestricted := withFreePass[tool]; !unrestricted {
			protected[tool] = struct{}{}
		}
	}
	return &RatchetGate{
		service:    service,
		sessions:   sessions,
		configured: configured,
		protected:  protected,
		lockSeed:   maphash.MakeSeed(),
		log:        log,
	}
}

// ValidateProtectedTools fails when any required tool is absent from the
// Ratchet rules or has a free-pass rule that makes its prerequisites optional.
func (g *RatchetGate) ValidateProtectedTools(required []string) error {
	missing := make([]string, 0)
	for _, name := range required {
		if _, ok := g.protected[ratchetdomain.ToolName(name)]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	slices.Sort(missing)
	return fmt.Errorf("ratchet policy leaves mutation tools unprotected: %s", strings.Join(missing, ", "))
}

// Middleware returns MCP receiving middleware that gates configured tool calls.
func (g *RatchetGate) Middleware() mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method != "tools/call" {
				return next(ctx, method, req)
			}

			params, ok := req.GetParams().(*mcp.CallToolParamsRaw)
			if !ok {
				return ratchetToolError(fmt.Errorf("ratchet: unexpected tools/call parameters %T", req.GetParams()))
			}
			tool := ratchetdomain.ToolName(params.Name)
			if _, governed := g.configured[tool]; !governed {
				return next(ctx, method, req)
			}

			session := req.GetSession()
			if session == nil || session.ID() == "" {
				return ratchetToolError(fmt.Errorf("ratchet: tool %q requires an MCP session", tool))
			}

			return g.execute(ctx, ratchetdomain.SessionID(session.ID()), tool, func() (mcp.Result, error) {
				return next(ctx, method, req)
			})
		}
	}
}

func (g *RatchetGate) execute(
	ctx context.Context,
	sessionID ratchetdomain.SessionID,
	tool ratchetdomain.ToolName,
	next func() (mcp.Result, error),
) (mcp.Result, error) {
	lockIndex := maphash.String(g.lockSeed, string(sessionID)) % uint64(len(g.locks))
	lock := &g.locks[lockIndex]
	lock.Lock()
	defer lock.Unlock()

	if _, err := g.service.CreateSession(ctx, sessionID); err != nil {
		g.warn("create session", tool, err)
		return ratchetToolError(fmt.Errorf("ratchet session: %w", err))
	}
	if err := g.service.ValidateToolCall(ctx, sessionID, tool, ""); err != nil {
		return ratchetToolError(fmt.Errorf("ratchet prerequisite: %w", err))
	}

	// Reserve a one-time prerequisite before executing the protected handler so
	// concurrent calls cannot both use the same token. A failed handler therefore
	// requires the caller to repeat its prerequisite before retrying.
	if err := g.service.ConsumePrerequisiteToken(ctx, sessionID, tool); err != nil {
		g.warn("consume prerequisite", tool, err)
		return ratchetToolError(fmt.Errorf("ratchet prerequisite consumption: %w", err))
	}

	result, err := next()
	if err != nil || toolResultFailed(result) {
		return result, err
	}

	session, err := g.sessions.Get(ctx, sessionID)
	if err != nil {
		g.warn("read session", tool, err)
		return ratchetToolError(fmt.Errorf("ratchet session read: %w", err))
	}
	session.RecordToolCall(tool)
	if err := g.sessions.Update(ctx, session); err != nil {
		g.warn("record tool call", tool, err)
		return ratchetToolError(fmt.Errorf("ratchet session update: %w", err))
	}
	if _, err := g.service.IssueToken(ctx, sessionID, tool); err != nil {
		g.warn("issue token", tool, err)
		return ratchetToolError(fmt.Errorf("ratchet token issuance: %w", err))
	}

	return result, nil
}

func (g *RatchetGate) warn(operation string, tool ratchetdomain.ToolName, err error) {
	if g.log != nil {
		g.log.Warn("ratchet enforcement failure", "operation", operation, "tool", tool, "err", err)
	}
}

func toolResultFailed(result mcp.Result) bool {
	toolResult, ok := result.(*mcp.CallToolResult)
	return ok && toolResult.IsError
}

func ratchetToolError(err error) (mcp.Result, error) {
	result := new(mcp.CallToolResult)
	result.SetError(err)
	return result, nil
}
