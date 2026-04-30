package services

import (
	"errors"
	"fmt"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

func resolveContextByteBudget(cmd *domain.LoadContextPackCommand) (int, error) {
	if cmd == nil {
		return 0, errors.New("context assembly: nil command")
	}
	if cmd.OmitBudget {
		if cmd.BudgetTokensApprox > 0 || cmd.MaxBudgetBytes > 0 || cmd.MaxTokens > 0 {
			return 0, errors.New("context assembly: omit_budget cannot be combined with other budget fields")
		}
		return maxContextMaxTokens, nil
	}

	if cmd.BudgetTokensApprox > 0 && (cmd.MaxBudgetBytes > 0 || cmd.MaxTokens > 0) {
		return 0, errors.New("context assembly: budget_tokens_approx conflicts with explicit byte budget (max_budget_bytes / legacy max_tokens)")
	}

	explicitBytes := cmd.MaxBudgetBytes
	if explicitBytes == 0 && cmd.MaxTokens > 0 && cmd.BudgetTokensApprox == 0 {
		// Legacy: callers or MCP mapped max_tokens → MaxTokens alone.
		explicitBytes = cmd.MaxTokens
	}

	switch {
	case cmd.BudgetTokensApprox > 0:
		budget, err := domain.ApproximateByteBudgetFromTokens(
			cmd.BudgetTokensApprox,
			cmd.BytesPerApproxToken,
			minContextMaxTokens,
			maxContextMaxTokens,
		)
		if err != nil {
			return 0, fmt.Errorf("context assembly: %w", err)
		}
		return budget, nil
	case explicitBytes > 0:
		return explicitBytes, nil
	default:
		return defaultContextMaxTokens, nil
	}
}
