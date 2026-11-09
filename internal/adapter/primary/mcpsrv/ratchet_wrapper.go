package mcpsrv

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hexxla/mcp-ratchet/pkg/ratchet"
	ratchetmcp "github.com/hexxla/mcp-ratchet/pkg/ratchet/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RatchetGate selects Mosaic policy tools; shared Ratchet middleware owns
// execution, prerequisite consumption, and final response serialization.
type RatchetGate struct {
	executor   ratchetmcp.Executor
	configured map[string]bool
}

// NewRatchetGate selects the tools governed by the supplied policy.
func NewRatchetGate(executor ratchetmcp.Executor, policy ratchet.Policy) *RatchetGate {
	configured := make(map[string]bool)
	unrestricted := make(map[string]bool)
	for _, rule := range policy.Rules {
		configured[rule.Tool] = len(rule.AfterAny) > 0
		if len(rule.AfterAny) == 0 {
			unrestricted[rule.Tool] = true
		}
		for _, prerequisite := range rule.AfterAny {
			if _, ok := configured[prerequisite]; !ok {
				configured[prerequisite] = false
			}
		}
	}
	for tool := range unrestricted {
		configured[tool] = false
	}
	return &RatchetGate{executor: executor, configured: configured}
}

// ValidateToolNames rejects policy references outside Mosaic's inventory.
func (g *RatchetGate) ValidateToolNames() error {
	var unknown []string
	for tool := range g.configured {
		if _, ok := mcpToolPolicies[tool]; !ok {
			unknown = append(unknown, tool)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	slices.Sort(unknown)
	return fmt.Errorf("ratchet policy names unknown Mosaic tools: %s", strings.Join(unknown, ", "))
}

// ValidateProtectedTools requires a guarded rule for every enabled mutation.
func (g *RatchetGate) ValidateProtectedTools(required []string) error {
	var missing []string
	for _, tool := range required {
		if !g.configured[tool] {
			missing = append(missing, tool)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	slices.Sort(missing)
	return fmt.Errorf("ratchet policy leaves mutation tools unprotected: %s", strings.Join(missing, ", "))
}

// PolicyForTools removes inactive guards and alternatives from an already
// name-validated policy. Enabled guards must retain an enabled prerequisite.
func PolicyForTools(policy ratchet.Policy, advanced bool) (ratchet.Policy, error) {
	enabled := make(map[string]bool)
	for _, tool := range EnabledToolNames(advanced) {
		enabled[tool] = true
	}
	filtered := ratchet.Policy{CompletionTTL: policy.CompletionTTL}
	for _, rule := range policy.Rules {
		if !enabled[rule.Tool] {
			continue
		}
		copy := rule
		copy.AfterAny = nil
		for _, prerequisite := range rule.AfterAny {
			if enabled[prerequisite] {
				copy.AfterAny = append(copy.AfterAny, prerequisite)
			}
		}
		if len(rule.AfterAny) > 0 && len(copy.AfterAny) == 0 {
			return ratchet.Policy{}, fmt.Errorf("ratchet policy tool %q has no enabled prerequisite", rule.Tool)
		}
		filtered.Rules = append(filtered.Rules, copy)
	}
	return filtered, nil
}

// Middleware delegates governed calls to the shared final-response middleware.
func (g *RatchetGate) Middleware() mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		governed := ratchetmcp.NewMiddleware(g.executor)(next)
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method != "tools/call" {
				return next(ctx, method, req)
			}
			call, ok := req.(*mcp.CallToolRequest)
			if !ok || call == nil || call.Params == nil {
				return governed(ctx, method, req)
			}
			if _, ok := g.configured[call.Params.Name]; !ok {
				return next(ctx, method, req)
			}
			return governed(ctx, method, req)
		}
	}
}
