package config

import (
	"fmt"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
)

// RetrievalBudgetConfig limits cumulative approximate LM tokens in JSON structured outputs
// from read MCP tools that return HexxlaDB-backed data, per MCP session (see MCP SDK session ID).
// SessionApproxTokenBudget 0 disables enforcement.
type RetrievalBudgetConfig struct {
	SessionApproxTokenBudget int
	// BytesPerApproxToken translates UTF-8 JSON byte length to approximate tokens (same range as context pack).
	// Zero means use [domain.DefaultApproxBytesPerToken] when constructing the tracker.
	BytesPerApproxToken float64
}

// DefaultRetrievalBudgetConfig is used when the YAML omits the retrieval section.
func DefaultRetrievalBudgetConfig() RetrievalBudgetConfig {
	return RetrievalBudgetConfig{}
}

func validateRetrievalBytesPerApproxToken(v float64) error {
	if v <= 0 {
		return nil
	}
	if v < float64(domain.MinApproxBytesPerToken) || v > float64(domain.MaxApproxBytesPerToken) {
		return fmt.Errorf("mosaic config: retrieval.bytes_per_approx_token must be between %d and %d inclusive (or omitted)",
			domain.MinApproxBytesPerToken, domain.MaxApproxBytesPerToken)
	}
	return nil
}
