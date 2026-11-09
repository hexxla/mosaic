package mcpsrv

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/config"
	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// RetrievalBudgetTracker enforces a per-MCP-session ceiling on approximate LM tokens charged against
// JSON-marshaled structured tool outputs from HexxlaDB read paths. Token cost is
// ceil(utf8_byte_length / bytes_per_approx_token), matching the context-pack byte/token story.
type RetrievalBudgetTracker struct {
	mu       sync.Mutex
	budget   int // 0 = unlimited
	bytesPer float64
	used     map[*mcp.ServerSession]int64
}

// NewRetrievalBudgetTracker builds a tracker from loaded Mosaic config. SessionApproxTokenBudget 0 disables enforcement.
func NewRetrievalBudgetTracker(cfg config.RetrievalBudgetConfig) *RetrievalBudgetTracker {
	per := cfg.BytesPerApproxToken
	if per <= 0 {
		per = float64(domain.DefaultApproxBytesPerToken)
	}
	if per < float64(domain.MinApproxBytesPerToken) {
		per = float64(domain.MinApproxBytesPerToken)
	}
	if per > float64(domain.MaxApproxBytesPerToken) {
		per = float64(domain.MaxApproxBytesPerToken)
	}
	return &RetrievalBudgetTracker{
		budget:   cfg.SessionApproxTokenBudget,
		bytesPer: per,
		used:     make(map[*mcp.ServerSession]int64),
	}
}

// Enabled reports whether session budgeting is active (positive budget).
func (t *RetrievalBudgetTracker) Enabled() bool {
	return t != nil && t.budget > 0
}

func budgetSession(req *mcp.CallToolRequest) *mcp.ServerSession {
	if req == nil || req.Session == nil || req.Session.ID() == "" {
		return nil
	}
	return req.Session
}

// RemainingBytes estimates the session's remaining response budget in bytes.
// Zero means no cap. The final serialized response still goes through metering.
func (t *RetrievalBudgetTracker) RemainingBytes(req *mcp.CallToolRequest) int {
	if !t.Enabled() {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	remaining := int64(t.budget) - t.used[budgetSession(req)]
	return int(min(float64(math.MaxInt/2), max(1, float64(remaining)*t.bytesPer)))
}

// BeforeRead returns an error when the session has already reached or exceeded the configured budget.
func (t *RetrievalBudgetTracker) BeforeRead(req *mcp.CallToolRequest) error {
	if t == nil || !t.Enabled() {
		return nil
	}
	session := budgetSession(req)
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.used[session] >= int64(t.budget) {
		return fmt.Errorf("mosaic retrieval budget exhausted: used %d >= budget %d approximate tokens",
			t.used[session], t.budget)
	}
	return nil
}

// RecordStructuredOutput meters approximate tokens for the JSON encoding of structuredOutput (always, when t != nil).
// When a positive session budget is configured, it returns an error without updating usage if this response would exceed the cap.
func (t *RetrievalBudgetTracker) RecordStructuredOutput(req *mcp.CallToolRequest, structuredOutput any) error {
	if t == nil {
		return nil
	}
	b, err := json.Marshal(structuredOutput)
	if err != nil {
		return fmt.Errorf("mosaic retrieval budget: marshal structured output: %w", err)
	}
	cost := approximateTokensFromJSONBytes(len(b), t.bytesPer)
	session := budgetSession(req)
	t.mu.Lock()
	defer t.mu.Unlock()
	next := t.used[session] + cost
	if t.budget > 0 && next > int64(t.budget) {
		return fmt.Errorf("mosaic retrieval budget would be exceeded: used %d + %d (this response) > budget %d approximate tokens",
			t.used[session], cost, t.budget)
	}
	if _, tracked := t.used[session]; !tracked && session != nil {
		go t.releaseOnClose(session)
	}
	t.used[session] = next
	return nil
}

func (t *RetrievalBudgetTracker) releaseOnClose(session *mcp.ServerSession) {
	_ = session.Wait()
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.used, session)
}

func approximateTokensFromJSONBytes(n int, bytesPer float64) int64 {
	if n <= 0 {
		return 0
	}
	return int64(math.Ceil(float64(n) / bytesPer))
}

// RunBudgetedRead runs fn, then records approximate retrieval tokens for out (metering).
// When enforcement is enabled (positive session budget), BeforeRead gates the call and RecordStructuredOutput rejects overshoots.
func (rb *RetrievalBudgetTracker) RunBudgetedRead[T any](req *mcp.CallToolRequest, fn func() (T, error)) (T, error) {
	if rb == nil {
		return fn()
	}
	if rb.Enabled() {
		if err := rb.BeforeRead(req); err != nil {
			var z T
			return z, err
		}
	}
	out, err := fn()
	if err != nil {
		var z T
		return z, err
	}
	if err := rb.RecordStructuredOutput(req, out); err != nil {
		var z T
		return z, err
	}
	return out, nil
}

// RetrievalBudgetStatus is the structured payload for mosaic_hexxla_retrieval_budget_status.
type RetrievalBudgetStatus struct {
	SessionApproxTokenBudget int     `json:"session_approx_token_budget"`
	ApproxTokensUsed         int64   `json:"approx_tokens_used"`
	BytesPerApproxToken      float64 `json:"bytes_per_approx_token"`
	SessionKey               string  `json:"session_key"`
	BudgetingEnabled         bool    `json:"budgeting_enabled"`
	// MeteringEnabled is true when the server records usage (always, for non-nil tracker). Same as tracking ApproxTokensUsed.
	MeteringEnabled bool `json:"metering_enabled"`
}

// Status snapshots usage for the MCP session attached to req.
func (t *RetrievalBudgetTracker) Status(req *mcp.CallToolRequest) RetrievalBudgetStatus {
	session := budgetSession(req)
	key := ""
	if session != nil {
		key = session.ID()
	}
	if t == nil {
		return RetrievalBudgetStatus{
			BytesPerApproxToken: float64(domain.DefaultApproxBytesPerToken),
			SessionKey:          key,
			MeteringEnabled:     false,
		}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	used := t.used[session]
	return RetrievalBudgetStatus{
		SessionApproxTokenBudget: t.budget,
		ApproxTokensUsed:         used,
		BytesPerApproxToken:      t.bytesPer,
		SessionKey:               key,
		BudgetingEnabled:         t.budget > 0,
		MeteringEnabled:          true,
	}
}

// RegisterRetrievalBudgetStatusTool registers mosaic_hexxla_retrieval_budget_status (read-only; never charged against the budget).
func RegisterRetrievalBudgetStatusTool(server *mcp.Server, tracker *RetrievalBudgetTracker, log *slog.Logger) error {
	return addTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_retrieval_budget_status",
		Description: "Answer: how much approximate retrieval (JSON from HexxlaDB read tools) this MCP session has accumulated. metering_enabled true when the server tracks usage. budgeting_enabled true when session_approx_token_budget > 0 (hard cap). When budgeting is off, usage is still metered for observability. Stateless clients without Mcp-Session-Id may share the empty session key.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, RetrievalBudgetStatus, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_retrieval_budget_status invoked")
		}
		return nil, tracker.Status(req), nil
	})
}
