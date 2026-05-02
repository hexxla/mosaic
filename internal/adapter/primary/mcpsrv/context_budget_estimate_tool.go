package mcpsrv

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	ratchetdomain "github.com/hexxla/mcp-ratchet/pkg/ratchet/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// contextBudgetEstimateOutput is the structured result for mosaic_hexxla_estimate_context_budget_bytes.
type contextBudgetEstimateOutput struct {
	BudgetBytes         int     `json:"budget_bytes"`
	TokensApprox        int     `json:"tokens_approx"`
	BytesPerApproxToken float64 `json:"bytes_per_approx_token_used"`
	MinClampBytes       int     `json:"min_clamp_bytes"`
	MaxClampBytes       int     `json:"max_clamp_bytes"`
}

type estimateInput struct {
	TokensApprox        int     `json:"tokens_approx"`
	BytesPerApproxToken float64 `json:"bytes_per_approx_token,omitempty"`
}

// RegisterContextBudgetEstimateTool registers mosaic_hexxla_estimate_context_budget_bytes (pure preview; same clamps as load_context_pack).
func RegisterContextBudgetEstimateTool(server *mcp.Server, log *slog.Logger, ratchetWrapper *RatchetWrapper) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, in estimateInput) (*mcp.CallToolResult, contextBudgetEstimateOutput, error) {
		return handleContextBudgetEstimate(ctx, req, &in, log)
	}

	// Wrap with ratchet if provided
	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, in estimateInput) (*mcp.CallToolResult, contextBudgetEstimateOutput, error) {
			sessionID := ratchetWrapper.DeriveSessionID(ctx)

			session, err := ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				session = ratchetdomain.NewSession(sessionID)
				if createErr := ratchetWrapper.sessionStore.Create(ctx, session); createErr != nil {
					if ratchetWrapper.log != nil {
						ratchetWrapper.log.WarnContext(ctx, "failed to create session", "error", createErr)
					}
				}
			}

			var token ratchetdomain.TokenValue
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_estimate_context_budget_bytes")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_estimate_context_budget_bytes"), token)
			if err != nil {
				return nil, contextBudgetEstimateOutput{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, in)
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_estimate_context_budget_bytes"))
			if err != nil {
				if ratchetWrapper.log != nil {
					ratchetWrapper.log.WarnContext(ctx, "failed to issue ratchet token", "error", err)
				}
			}

			session, err = ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				if ratchetWrapper.log != nil {
					ratchetWrapper.log.WarnContext(ctx, "failed to get session after token issuance", "error", err)
				}
			} else {
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_estimate_context_budget_bytes"))
				if updateErr := ratchetWrapper.sessionStore.Update(ctx, session); updateErr != nil {
					if ratchetWrapper.log != nil {
						ratchetWrapper.log.WarnContext(ctx, "failed to update session with tool call", "error", updateErr)
					}
				}
			}

			return result, resp, nil
		}
		handler = wrappedHandler
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_estimate_context_budget_bytes",
		Description: "Preview UTF-8 byte budget from an approximate token count (same approximation and clamps as mosaic_hexxla_load_context_pack with budget_tokens_approx). Call before load_context_pack when tuning budget_tokens_approx or max_budget_bytes. Hexxla counts UTF-8 bytes (ByteLenBudgeter), not tokenizer tokens.",
	}, handler)
}

func handleContextBudgetEstimate(ctx context.Context, _ *mcp.CallToolRequest, in *estimateInput, log *slog.Logger) (*mcp.CallToolResult, contextBudgetEstimateOutput, error) {
	if log != nil {
		log.DebugContext(ctx, "mosaic_hexxla_estimate_context_budget_bytes invoked")
	}
	if in.TokensApprox <= 0 {
		return nil, contextBudgetEstimateOutput{}, errors.New("tokens_approx must be positive")
	}
	perDisplayed := float64(domain.DefaultApproxBytesPerToken)
	if in.BytesPerApproxToken > 0 {
		perDisplayed = in.BytesPerApproxToken
	}
	out := contextBudgetEstimateOutput{
		TokensApprox:        in.TokensApprox,
		BytesPerApproxToken: perDisplayed,
		MinClampBytes:       domain.ContextBudgetMinBytes,
		MaxClampBytes:       domain.ContextBudgetMaxBytes,
	}
	budget, err := domain.ApproximateByteBudgetFromTokens(
		in.TokensApprox,
		in.BytesPerApproxToken,
		domain.ContextBudgetMinBytes,
		domain.ContextBudgetMaxBytes,
	)
	if err != nil {
		return nil, contextBudgetEstimateOutput{}, err
	}
	out.BudgetBytes = budget
	return nil, out, nil
}
