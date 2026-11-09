package mcpsrv

import (
	"context"
	"fmt"
	"slices"
	"strings"

	ratchetdomain "github.com/hexxla/mcp-ratchet/pkg/ratchet/domain"
	ratchetprimary "github.com/hexxla/mcp-ratchet/pkg/ratchet/ports/primary"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RatchetGate enforces configured tool prerequisites for each MCP session.
type RatchetGate struct {
	executor   ratchetprimary.WorkflowExecutor
	configured map[ratchetdomain.ToolName]bool
}

// NewRatchetGate wires Ratchet's executor into the MCP tool boundary.
func NewRatchetGate(executor ratchetprimary.WorkflowExecutor, rules []ratchetdomain.Rule) *RatchetGate {
	configured := make(map[ratchetdomain.ToolName]bool, len(rules))
	unrestricted := make(map[ratchetdomain.ToolName]struct{})
	for _, rule := range rules {
		configured[rule.Tool] = rule.Prerequisite != ""
		if rule.Prerequisite == "" {
			unrestricted[rule.Tool] = struct{}{}
		}
		if rule.Prerequisite != "" {
			if _, ok := configured[rule.Prerequisite]; !ok {
				configured[rule.Prerequisite] = false
			}
		}
	}
	for tool := range unrestricted {
		configured[tool] = false
	}
	return &RatchetGate{executor: executor, configured: configured}
}

// ValidateToolNames rejects policy entries outside Mosaic's tool inventory.
func (g *RatchetGate) ValidateToolNames() error {
	var unknown []string
	for tool := range g.configured {
		if _, ok := mcpToolPolicies[string(tool)]; !ok {
			unknown = append(unknown, string(tool))
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	slices.Sort(unknown)
	return fmt.Errorf("ratchet policy names unknown Mosaic tools: %s", strings.Join(unknown, ", "))
}

// ValidateProtectedTools fails when any required tool is absent from the
// Ratchet rules or has a free-pass rule that makes its prerequisites optional.
func (g *RatchetGate) ValidateProtectedTools(required []string) error {
	missing := make([]string, 0)
	for _, name := range required {
		if !g.configured[ratchetdomain.ToolName(name)] {
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
	var result mcp.Result
	var handlerErr error
	err := g.executor.ExecuteToolCall(ctx, sessionID, tool, func() (bool, error) {
		result, handlerErr = next()
		return handlerErr == nil && !toolResultFailed(result), handlerErr
	})
	if handlerErr != nil {
		return result, handlerErr
	}
	if err != nil {
		return ratchetToolError(err)
	}
	return result, nil
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
