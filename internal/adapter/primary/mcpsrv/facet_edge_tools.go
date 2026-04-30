package mcpsrv

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
)

// RegisterFacetEdgeTools registers mosaic_hexxla_put_facet and mosaic_hexxla_link_cells.
func RegisterFacetEdgeTools(server *mcp.Server, svc primary.FacetEdge, log *slog.Logger) {
	registerPutFacetTool(server, svc, log)
	registerLinkCellsTool(server, svc, log)
}

func registerPutFacetTool(server *mcp.Server, svc primary.FacetEdge, log *slog.Logger) {
	type putFacetInput struct {
		Q              int    `json:"q" jsonschema:"axial q of host cell"`
		R              int    `json:"r" jsonschema:"axial r"`
		FacetID        byte   `json:"facet_id" jsonschema:"0..5 facet slot"`
		DerivedContent string `json:"derived_content" jsonschema:"facet body (direct PutFacet)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_put_facet",
		Description: "Write derived facet content to slot facet_id (0..5) at Hexxla cell (q,r) via Tx.PutFacet. Does not enforce derivation-hash coupling (use engine UpdateFacet in native code when you need content-hash guarding).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in putFacetInput) (*mcp.CallToolResult, domain.MutationOK, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_put_facet invoked", "q", in.Q, "r", in.R, "facet_id", in.FacetID)
		}
		err := svc.PutFacet(ctx, &domain.PutFacetCommand{
			Coord:          domain.AxialCoord{Q: in.Q, R: in.R},
			FacetID:        in.FacetID,
			DerivedContent: in.DerivedContent,
		})
		if err != nil {
			return nil, domain.MutationOK{}, err
		}
		return nil, domain.MutationOK{OK: true}, nil
	})
}

func registerLinkCellsTool(server *mcp.Server, svc primary.FacetEdge, log *slog.Logger) {
	type linkInput struct {
		FromQ        int      `json:"from_q"`
		FromR        int      `json:"from_r"`
		ToQ          int      `json:"to_q"`
		ToR          int      `json:"to_r"`
		RelationType string   `json:"relation_type" jsonschema:"non-empty relation label"`
		Weight       *float64 `json:"weight,omitempty" jsonschema:"edge weight (default 1.0)"`
		SourceID     string   `json:"source_id" jsonschema:"provenance source id"`
		Confidence   *float64 `json:"confidence,omitempty" jsonschema:"0..1 (default 1.0 if omitted)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_link_cells",
		Description: "Create or replace an edge between two axial cells Tx.LinkCells → Tx.PutEdge (relation_type, weight, provenance timestamps set to now). Direction matters for graph traversals AscendEdgesFrom(from).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in linkInput) (*mcp.CallToolResult, domain.MutationOK, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_link_cells invoked")
		}
		w := 1.0
		if in.Weight != nil {
			w = *in.Weight
		}
		c := 1.0
		if in.Confidence != nil {
			c = *in.Confidence
		}
		err := svc.LinkCells(ctx, &domain.LinkCellsCommand{
			From:         domain.AxialCoord{Q: in.FromQ, R: in.FromR},
			To:           domain.AxialCoord{Q: in.ToQ, R: in.ToR},
			RelationType: in.RelationType,
			Weight:       w,
			SourceID:     in.SourceID,
			Confidence:   c,
		})
		if err != nil {
			return nil, domain.MutationOK{}, err
		}
		return nil, domain.MutationOK{OK: true}, nil
	})
}
