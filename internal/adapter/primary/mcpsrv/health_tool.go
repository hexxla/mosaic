package mcpsrv

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/ports/primary"
)

// RegisterHealthTool registers the mosaic_hexxla_health MCP tool, which delegates to [primary.Health].
func RegisterHealthTool(server *mcp.Server, health primary.Health, log *slog.Logger, budget *RetrievalBudgetTracker) {
	type healthInput struct{} // No parameters.

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_health",
		Description: "Run HexxlaDB HealthCheck (cells, seams, tag/source indexes, orphans, MVCC stats, warnings) plus database layout (page size, max value bytes, embedding dimension/metric) and integrity_ok. Use before heavy retrieval when embedding setup or DB integrity is unknown; empty embedding hits may mean dimension mismatch or missing vectors.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ healthInput) (*mcp.CallToolResult, domain.HealthSummary, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_health invoked")
		}
		out, err := RunBudgetedRead(budget, req, func() (domain.HealthSummary, error) {
			return health.Status(ctx)
		})
		if err != nil {
			return nil, domain.HealthSummary{}, err
		}
		return nil, out, nil
	})
}
