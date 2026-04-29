package mcpsrv

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/ports/primary"
)

// RegisterSeamTools wires seam discovery and writes: find_seams, mark_conflict, mark_supersedes, resolve_seam.
func RegisterSeamTools(server *mcp.Server, svc primary.SeamLifecycle, log *slog.Logger, budget *RetrievalBudgetTracker) {
	registerFindSeamsTool(server, svc, log, budget)
	registerMarkConflictTool(server, svc, log)
	registerMarkSupersedesTool(server, svc, log)
	registerResolveSeamTool(server, svc, log)
}

func registerFindSeamsTool(server *mcp.Server, svc primary.SeamLifecycle, log *slog.Logger, budget *RetrievalBudgetTracker) {
	type findSeamsInput struct {
		CenterQ        int  `json:"center_q" jsonschema:"axial q of search center"`
		CenterR        int  `json:"center_r" jsonschema:"axial r of search center"`
		Radius         *int `json:"radius,omitempty" jsonschema:"hex rings from center; omit for default 3, use 0 for center cell only"`
		UnresolvedOnly bool `json:"unresolved_only,omitempty" jsonschema:"only seams with empty resolution_status"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_find_seams",
		Description: "List seams HexxlaDB.FindSeams: endpoints within hex distance radius of (center_q, center_r). Set unresolved_only to focus on seams not yet ResolveSeam'd. Larger radii scan more neighbouring cells.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in findSeamsInput) (*mcp.CallToolResult, domain.FindSeamsResponse, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_find_seams invoked", "center_q", in.CenterQ, "center_r", in.CenterR)
		}
		radius := 3
		if in.Radius != nil {
			radius = *in.Radius
		}
		out, err := RunBudgetedRead(budget, req, func() (domain.FindSeamsResponse, error) {
			return svc.FindSeams(ctx, &domain.FindSeamsQuery{
				Center:         domain.AxialCoord{Q: in.CenterQ, R: in.CenterR},
				Radius:         radius,
				UnresolvedOnly: in.UnresolvedOnly,
			})
		})
		if err != nil {
			return nil, domain.FindSeamsResponse{}, err
		}
		return nil, out, nil
	})
}

func registerMarkConflictTool(server *mcp.Server, svc primary.SeamLifecycle, log *slog.Logger) {
	type markConflictInput struct {
		Aq     int    `json:"aq" jsonschema:"cell A axial q"`
		Ar     int    `json:"ar" jsonschema:"cell A axial r"`
		Bq     int    `json:"bq" jsonschema:"cell B axial q"`
		Br     int    `json:"br" jsonschema:"cell B axial r"`
		Reason string `json:"reason,omitempty" jsonschema:"human-readable conflict reason"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_mark_conflict",
		Description: "Create a contradiction seam Tx.MarkConflict between two axial cells — canonical endpoints, SeamType mark_conflict — for review or resolution workflows.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in markConflictInput) (*mcp.CallToolResult, domain.MutationOK, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_mark_conflict invoked")
		}
		err := svc.MarkConflict(ctx,
			domain.AxialCoord{Q: in.Aq, R: in.Ar},
			domain.AxialCoord{Q: in.Bq, R: in.Br},
			in.Reason,
		)
		if err != nil {
			return nil, domain.MutationOK{}, err
		}
		return nil, domain.MutationOK{OK: true}, nil
	})
}

func registerMarkSupersedesTool(server *mcp.Server, svc primary.SeamLifecycle, log *slog.Logger) {
	type markSupInput struct {
		SupersederQ int    `json:"superseder_q" jsonschema:"current-truth cell q"`
		SupersederR int    `json:"superseder_r" jsonschema:"current-truth cell r"`
		SupersededQ int    `json:"superseded_q" jsonschema:"superseded (stale) cell q"`
		SupersededR int    `json:"superseded_r" jsonschema:"superseded cell r"`
		Reason      string `json:"reason,omitempty"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_mark_supersedes",
		Description: "Record supersession Tx.MarkSupersedes: superseder_* is current truth, superseded_* is stale. SeamType supersedes — used by mosaic_hexxla_load_context_pack when FilterSuperseded is enabled.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in markSupInput) (*mcp.CallToolResult, domain.MutationOK, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_mark_supersedes invoked")
		}
		err := svc.MarkSupersedes(ctx,
			domain.AxialCoord{Q: in.SupersederQ, R: in.SupersederR},
			domain.AxialCoord{Q: in.SupersededQ, R: in.SupersededR},
			in.Reason,
		)
		if err != nil {
			return nil, domain.MutationOK{}, err
		}
		return nil, domain.MutationOK{OK: true}, nil
	})
}

func registerResolveSeamTool(server *mcp.Server, svc primary.SeamLifecycle, log *slog.Logger) {
	type resolveInput struct {
		SeamID           string `json:"seam_id" jsonschema:"26-char ULID from find_seams"`
		ResolutionStatus string `json:"resolution_status" jsonschema:"non-empty status label"`
		ResolutionNote   string `json:"resolution_note,omitempty"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_resolve_seam",
		Description: "Update resolution fields on an existing seam Tx.ResolveSeam. Fails with seam not found if the id is unknown.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in resolveInput) (*mcp.CallToolResult, domain.MutationOK, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_resolve_seam invoked")
		}
		err := svc.ResolveSeam(ctx, in.SeamID, in.ResolutionStatus, in.ResolutionNote)
		if err != nil {
			return nil, domain.MutationOK{}, err
		}
		return nil, domain.MutationOK{OK: true}, nil
	})
}
