package primary

import (
	"context"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
)

// Health is the driving port for retrieving a database health summary (MCP tools call into this).
type Health interface {
	Status(ctx context.Context) (domain.HealthSummary, error)
}
