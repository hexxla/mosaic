package mcpsrv

import (
	"context"
	"errors"
	"log/slog"

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

// RegisterContextBudgetEstimateTool registers mosaic_hexxla_estimate_context_budget_bytes (pure preview; same clamps as load_context_pack).
func RegisterContextBudgetEstimateTool(server *mcp.Server, log *slog.Logger) error {
	type estimateInput struct {
		TokensApprox        int     `json:"tokens_approx"`
		BytesPerApproxToken float64 `json:"bytes_per_approx_token,omitempty"`
	}

	return addTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_estimate_context_budget_bytes",
		Description: "Convert an approximate token count to a UTF-8 byte budget, clamped to 64-100000 bytes. Pass budget_bytes as load_context_pack max_budget_bytes. This estimate does not use a provider tokenizer.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in estimateInput) (*mcp.CallToolResult, contextBudgetEstimateOutput, error) {
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
	})
}
