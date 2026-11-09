package mcpsrv

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
)

// RegisterHealthTool registers the mosaic_hexxla_health MCP tool, which delegates to [primary.Health].
func RegisterHealthTool(server *mcp.Server, health primary.Health, log *slog.Logger, budget *RetrievalBudgetTracker) error {
	type healthInput struct {
		FullCheck bool `json:"full_check,omitempty" jsonschema:"run the full integrity scan (can block writes)"`
	}

	return addTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_health",
		Description: "Report database layout, embedding configuration, disk footprint, and retention policy. Pass full_check=true to scan cells, seams, and tag/source indexes. Integrity results and scan counts apply only when integrity_checked is true.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in healthInput) (*mcp.CallToolResult, domain.HealthSummary, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_health invoked")
		}
		out, err := budget.RunBudgetedRead(req, func() (domain.HealthSummary, error) {
			return health.Status(ctx, in.FullCheck)
		})
		if err != nil {
			return nil, domain.HealthSummary{}, err
		}
		return nil, out, nil
	})
}
