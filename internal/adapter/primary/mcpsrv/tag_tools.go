package mcpsrv

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
)

// RegisterListTagsTool registers the core tag-vocabulary tool.
func RegisterListTagsTool(server *mcp.Server, svc primary.TagBrowse, log *slog.Logger, budget *RetrievalBudgetTracker) error {
	return addTool(server, &mcp.Tool{
		Name: "mosaic_hexxla_list_tags",
		Description: "List all distinct cell tags visible in the DB (sorted), same semantics as Hexxla Tx.ListExistingTopics — read-only. " +
			"Call before mosaic_hexxla_put_cell whenever tags might overlap existing vocabulary; reuse tags from this list when they fit. " +
			"When lexical search misses, align queries with vocabulary from here.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, domain.DistinctTagsResult, error) {
		if log != nil {
			log.Debug("mcp mosaic_hexxla_list_tags")
		}
		out, err := budget.RunBudgetedRead(req, func() (domain.DistinctTagsResult, error) {
			return svc.ListDistinctTags(ctx)
		})
		if err != nil {
			return nil, domain.DistinctTagsResult{}, err
		}
		return nil, out, nil
	})
}

// RegisterTagCountsTool registers advanced tag-frequency browsing.
func RegisterTagCountsTool(server *mcp.Server, svc primary.TagBrowse, log *slog.Logger, budget *RetrievalBudgetTracker) error {
	return addTool(server, &mcp.Tool{
		Name: "mosaic_hexxla_tag_counts",
		Description: "Per-tag visible cell counts (sorted high-to-low frequency), Hexxla Tx.TagCounts semantics — read-only. " +
			"Use with mosaic_hexxla_list_tags before put_cell to prefer high-frequency established tags over one-off spellings. Also narrows search/query filters.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, domain.TagCountsResult, error) {
		if log != nil {
			log.Debug("mcp mosaic_hexxla_tag_counts")
		}
		out, err := budget.RunBudgetedRead(req, func() (domain.TagCountsResult, error) {
			return svc.TagCounts(ctx)
		})
		if err != nil {
			return nil, domain.TagCountsResult{}, err
		}
		return nil, out, nil
	})
}
