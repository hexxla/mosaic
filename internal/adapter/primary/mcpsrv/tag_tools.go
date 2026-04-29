package mcpsrv

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/ports/primary"
)

// RegisterTagBrowseTools registers read-only mosaic_hexxla_list_tags and mosaic_hexxla_tag_counts.
func RegisterTagBrowseTools(server *mcp.Server, svc primary.TagBrowse, log *slog.Logger, budget *RetrievalBudgetTracker) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_list_tags",
		Description: "List all distinct cell tags visible in the DB (sorted), same semantics as Hexxla Tx.ListExistingTopics — read-only. Use BEFORE mosaic_hexxla_put_cell to reuse existing taxonomy; when lexical search misses, broaden or align queries with vocabulary from here.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, domain.DistinctTagsResult, error) {
		if log != nil {
			log.Debug("mcp mosaic_hexxla_list_tags")
		}
		out, err := RunBudgetedRead(budget, req, func() (domain.DistinctTagsResult, error) {
			return svc.ListDistinctTags(ctx)
		})
		if err != nil {
			return nil, domain.DistinctTagsResult{}, err
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_tag_counts",
		Description: "Per-tag visible cell counts (sorted high-to-low frequency), Hexxla Tx.TagCounts semantics — read-only. Helps prioritize common tags vs rare ones when choosing tags for mosaic_hexxla_put_cell or narrowing search/query.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, domain.TagCountsResult, error) {
		if log != nil {
			log.Debug("mcp mosaic_hexxla_tag_counts")
		}
		out, err := RunBudgetedRead(budget, req, func() (domain.TagCountsResult, error) {
			return svc.TagCounts(ctx)
		})
		if err != nil {
			return nil, domain.TagCountsResult{}, err
		}
		return nil, out, nil
	})
}
