package mcpsrv

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/ports/primary"
)

// RegisterFacetEdgeBrowseTools registers read-only facet and edge lookup tools — View-only, no mutations.
func RegisterFacetEdgeBrowseTools(server *mcp.Server, svc primary.FacetEdgeBrowse, log *slog.Logger, budget *RetrievalBudgetTracker) {
	registerGetFacet(server, svc, log, budget)
	registerListFacets(server, svc, log, budget)
	registerGetEdge(server, svc, log, budget)
	registerListEdgesFrom(server, svc, log, budget)
}

func registerGetFacet(server *mcp.Server, svc primary.FacetEdgeBrowse, log *slog.Logger, budget *RetrievalBudgetTracker) {
	type input struct {
		Q       int  `json:"q"`
		R       int  `json:"r"`
		FacetID byte `json:"facet_id" jsonschema:"0..5 facet slot"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_get_facet",
		Description: "Read one facet slot at (q,r) via Tx.GetFacet — read-only snapshot. Pair with mosaic_hexxla_put_facet verification before overwriting.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in input) (*mcp.CallToolResult, domain.GetFacetResponse, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_get_facet")
		}
		out, err := RunBudgetedRead(budget, req, func() (domain.GetFacetResponse, error) {
			return svc.GetFacet(ctx, domain.AxialCoord{Q: in.Q, R: in.R}, in.FacetID)
		})
		return nil, out, err
	})
}

func registerListFacets(server *mcp.Server, svc primary.FacetEdgeBrowse, log *slog.Logger, budget *RetrievalBudgetTracker) {
	type input struct {
		Q int `json:"q"`
		R int `json:"r"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_list_facets",
		Description: "List all visible facets at (q,r) via Tx.AscendFacetsForCell — read-only snapshot (slots returned in engine order).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in input) (*mcp.CallToolResult, domain.ListFacetsForCellResponse, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_list_facets")
		}
		out, err := RunBudgetedRead(budget, req, func() (domain.ListFacetsForCellResponse, error) {
			return svc.ListFacetsForCell(ctx, domain.AxialCoord{Q: in.Q, R: in.R})
		})
		return nil, out, err
	})
}

func registerGetEdge(server *mcp.Server, svc primary.FacetEdgeBrowse, log *slog.Logger, budget *RetrievalBudgetTracker) {
	type input struct {
		FromQ        int    `json:"from_q"`
		FromR        int    `json:"from_r"`
		ToQ          int    `json:"to_q"`
		ToR          int    `json:"to_r"`
		RelationType string `json:"relation_type" jsonschema:"exact edge key label"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_get_edge",
		Description: "Read one directed edge (from_cell → to_cell, relation_type) via Tx.GetEdge — read-only. Direction matches mosaic_hexxla_link_cells / AscendEdgesFrom(from).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in input) (*mcp.CallToolResult, domain.GetEdgeResponse, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_get_edge")
		}
		out, err := RunBudgetedRead(budget, req, func() (domain.GetEdgeResponse, error) {
			return svc.GetEdge(ctx,
				domain.AxialCoord{Q: in.FromQ, R: in.FromR},
				domain.AxialCoord{Q: in.ToQ, R: in.ToR},
				in.RelationType,
			)
		})
		return nil, out, err
	})
}

func registerListEdgesFrom(server *mcp.Server, svc primary.FacetEdgeBrowse, log *slog.Logger, budget *RetrievalBudgetTracker) {
	type input struct {
		FromQ    int `json:"from_q"`
		FromR    int `json:"from_r"`
		MaxEdges int `json:"max_edges,omitempty" jsonschema:"optional cap default 50 max 200; truncated flag if more existed"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_list_edges_from",
		Description: "List outbound edges whose from-cell equals (from_q,from_r) via Tx.AscendEdgesFrom — read-only. Bounded by max_edges (default 50, cap 200); truncated=true when scan stopped early at the cap.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in input) (*mcp.CallToolResult, domain.ListEdgesFromResponse, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_list_edges_from")
		}
		out, err := RunBudgetedRead(budget, req, func() (domain.ListEdgesFromResponse, error) {
			return svc.ListEdgesFrom(ctx, domain.AxialCoord{Q: in.FromQ, R: in.FromR}, in.MaxEdges)
		})
		return nil, out, err
	})
}
