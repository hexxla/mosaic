package mcpsrv

import (
	"context"
	"errors"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
)

type seedCoord struct {
	Q int `json:"q" jsonschema:"axial q"`
	R int `json:"r" jsonschema:"axial r"`
}

type contextPackInput struct {
	Snapshot            *domain.SnapshotSelector `json:"snapshot,omitempty" jsonschema:"optional retained snapshot: exactly one of seq, tag or RFC3339 at; historical edges/vectors unavailable"`
	Seeds               []seedCoord              `json:"seeds" jsonschema:"one or more lattice coords from get_cell or search_cells"`
	EdgeFilter          string                   `json:"edge_filter,omitempty" jsonschema:"comma-separated relations; selects graph BFS instead of radial expansion"`
	MaxHops             int                      `json:"max_hops,omitzero" jsonschema:"graph depth, default 5, maximum 32"`
	AsOf                string                   `json:"as_of,omitempty" jsonschema:"RFC3339 validity time; distinct from historical snapshot time"`
	IncludeEdges        bool                     `json:"include_edges,omitzero"`
	SeamRadius          int                      `json:"seam_radius,omitzero" jsonschema:"regional seam radius, default max_ring, maximum 32"`
	UnresolvedSeamsOnly bool                     `json:"unresolved_seams_only,omitzero"`
	MaxRing             int                      `json:"max_ring,omitzero" jsonschema:"hex rings from each seed (default 3, max 32)"`
	MaxCells            int                      `json:"max_cells,omitzero" jsonschema:"candidate pool cap before eviction (default 256, max 8192)"`
	MaxBudgetBytes      int                      `json:"max_budget_bytes,omitzero" jsonschema:"UTF-8 byte ceiling (64-100000, default 4096)"`

	FilterSuperseded bool `json:"filter_superseded,omitempty"`
	IncludeSeams     bool `json:"include_seams,omitempty"`
	IncludeFacetText bool `json:"include_facet_text,omitempty" jsonschema:"include facet-derived text in each cell and count it toward the byte budget"`
	Explain          bool `json:"explain,omitempty"`
}

func validateContextPackBudgetInput(in *contextPackInput) error {
	if in == nil {
		return errors.New("context pack budget: nil input")
	}
	if in.MaxBudgetBytes != 0 && (in.MaxBudgetBytes < domain.ContextBudgetMinBytes || in.MaxBudgetBytes > domain.ContextBudgetMaxBytes) {
		return errors.New("context pack budget: max_budget_bytes must be between 64 and 100000")
	}
	return nil
}

// RegisterContextPackTool registers mosaic_hexxla_load_context_pack.
func RegisterContextPackTool(server *mcp.Server, svc primary.ContextAssembly, log *slog.Logger, budget *RetrievalBudgetTracker) error {
	return addTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_load_context_pack",
		Description: "Expand lattice context from get_cell or search_cells coordinates. Mosaic enforces max_budget_bytes (default 4096, maximum 100000 UTF-8 bytes), evicting outer-ring and lower-confidence cells first. Optional seams and facet text provide neighbouring context.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in contextPackInput) (*mcp.CallToolResult, domain.ContextPackResponse, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_load_context_pack invoked")
		}
		if len(in.Seeds) == 0 {
			return nil, domain.ContextPackResponse{}, errors.New("seeds: at least one {q,r} required")
		}
		if err := validateContextPackBudgetInput(&in); err != nil {
			return nil, domain.ContextPackResponse{}, err
		}
		seeds := make([]domain.AxialCoord, 0, len(in.Seeds))
		for _, s := range in.Seeds {
			seeds = append(seeds, domain.AxialCoord{Q: s.Q, R: s.R})
		}
		asOf, err := parseOptionalRFC3339(in.AsOf)
		if err != nil {
			return nil, domain.ContextPackResponse{}, err
		}
		cmd := &domain.LoadContextPackCommand{Snapshot: in.Snapshot,
			EdgeFilter: in.EdgeFilter, MaxHops: in.MaxHops, AsOf: asOf, IncludeEdges: in.IncludeEdges, SeamRadius: in.SeamRadius, UnresolvedSeamsOnly: in.UnresolvedSeamsOnly,
			Seeds:            seeds,
			MaxRing:          in.MaxRing,
			MaxBudgetBytes:   in.MaxBudgetBytes,
			MaxCells:         in.MaxCells,
			FilterSuperseded: in.FilterSuperseded,
			IncludeSeams:     in.IncludeSeams,
			IncludeFacetText: in.IncludeFacetText,
			Explain:          in.Explain,
		}
		out, err := budget.RunBudgetedRead(req, func() (domain.ContextPackResponse, error) {
			cmd.SessionByteLimit = budget.RemainingBytes(req)
			return svc.LoadFromSeeds(ctx, cmd)
		})
		if err != nil {
			return nil, domain.ContextPackResponse{}, err
		}
		return nil, out, nil
	})
}
