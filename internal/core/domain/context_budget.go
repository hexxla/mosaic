package domain

import (
	"errors"
	"fmt"
	"math"
)

// Defaults for translating approximate LM token budgets to the UTF-8 byte
// budgets enforced by Mosaic. No provider tokenizer is required.
const (
	DefaultApproxBytesPerToken = 4
	MinApproxBytesPerToken     = 2
	MaxApproxBytesPerToken     = 16
)

// ContextByteBudgetBounds are Mosaic clamps applied to resolved UTF-8 byte budgets for
// Mosaic context assembly.
const (
	ContextBudgetMinBytes = 64
	ContextBudgetMaxBytes = 100000
)

// ApproximateByteBudgetFromTokens converts an approximate token count to an integer byte
// budget via bytesPerApproxToken (default [DefaultApproxBytesPerToken]). Result is capped
// to [minClamp, maxClamp] so it matches Mosaic context-assembly clamps.
func ApproximateByteBudgetFromTokens(tokens int, bytesPerApproxToken float64, minClamp, maxClamp int) (int, error) {
	if tokens <= 0 {
		return 0, errors.New("approximate byte budget: tokens must be positive")
	}
	per := float64(DefaultApproxBytesPerToken)
	if bytesPerApproxToken > 0 {
		per = bytesPerApproxToken
	}
	if per < float64(MinApproxBytesPerToken) || per > float64(MaxApproxBytesPerToken) {
		return 0, fmt.Errorf("approximate byte budget: bytes_per_approx_token must be between %d and %d inclusive",
			MinApproxBytesPerToken, MaxApproxBytesPerToken)
	}
	budget := max(int(math.Ceil(float64(tokens)*per)), minClamp)
	budget = min(budget, maxClamp)
	return budget, nil
}
