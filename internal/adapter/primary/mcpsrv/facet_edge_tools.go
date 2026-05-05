package mcpsrv

import (
	"context"
	"fmt"
	"log/slog"

	ratchetdomain "github.com/hexxla/mcp-ratchet/pkg/ratchet/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
)

type putFacetInput struct {
	Q              int    `json:"q" jsonschema:"axial q of host cell"`
	R              int    `json:"r" jsonschema:"axial r"`
	FacetID        byte   `json:"facet_id" jsonschema:"0..5 facet slot"`
	DerivedContent string `json:"derived_content" jsonschema:"facet body (direct PutFacet)"`
}

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

// RegisterFacetEdgeTools registers mosaic_hexxla_put_facet and mosaic_hexxla_link_cells.
func RegisterFacetEdgeTools(server *mcp.Server, svc primary.FacetEdge, log *slog.Logger, ratchetWrapper *RatchetWrapper) {
	registerPutFacetTool(server, svc, log, ratchetWrapper)
	registerLinkCellsTool(server, svc, log, ratchetWrapper)
}

func registerPutFacetTool(server *mcp.Server, svc primary.FacetEdge, log *slog.Logger, ratchetWrapper *RatchetWrapper) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, in putFacetInput) (*mcp.CallToolResult, domain.MutationOK, error) {
		return handlePutFacet(ctx, req, &in, svc, log)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, in putFacetInput) (*mcp.CallToolResult, domain.MutationOK, error) {
			sessionID := ratchetWrapper.DeriveSessionID(ctx)

			// Use ratchetSvc.CreateSession for session_created events
			_, err := ratchetWrapper.ratchetSvc.CreateSession(ctx, sessionID)
			if err != nil {
				return nil, domain.MutationOK{}, fmt.Errorf("failed to create session: %w", err)
			}

			session, err := ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				return nil, domain.MutationOK{}, fmt.Errorf("failed to get session: %w", err)
			}

			var token ratchetdomain.TokenValue
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_put_facet")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_put_facet"), token)
			if err != nil {
				return nil, domain.MutationOK{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, in)
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_put_facet"))
			if err != nil {
				if ratchetWrapper.log != nil {
					ratchetWrapper.log.WarnContext(ctx, "failed to issue ratchet token", "error", err)
				}
			}

			session, err = ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				if ratchetWrapper.log != nil {
					ratchetWrapper.log.WarnContext(ctx, "failed to get session after token issuance", "error", err)
				}
			} else {
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_put_facet"))
				if updateErr := ratchetWrapper.sessionStore.Update(ctx, session); updateErr != nil {
					if ratchetWrapper.log != nil {
						ratchetWrapper.log.WarnContext(ctx, "failed to update session with tool call", "error", updateErr)
					}
				}
			}

			return result, resp, nil
		}
		handler = wrappedHandler
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_put_facet",
		Description: "Write derived facet content to slot facet_id (0..5) at Hexxla cell (q,r) via Tx.PutFacet. Does not enforce derivation-hash coupling (use engine UpdateFacet in native code when you need content-hash guarding).",
	}, handler)
}

func handlePutFacet(ctx context.Context, _ *mcp.CallToolRequest, in *putFacetInput, svc primary.FacetEdge, log *slog.Logger) (*mcp.CallToolResult, domain.MutationOK, error) {
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
}

func registerLinkCellsTool(server *mcp.Server, svc primary.FacetEdge, log *slog.Logger, ratchetWrapper *RatchetWrapper) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, in linkInput) (*mcp.CallToolResult, domain.MutationOK, error) {
		return handleLinkCells(ctx, req, &in, svc, log)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, in linkInput) (*mcp.CallToolResult, domain.MutationOK, error) {
			sessionID := ratchetWrapper.DeriveSessionID(ctx)

			// Use ratchetSvc.CreateSession for session_created events
			_, err := ratchetWrapper.ratchetSvc.CreateSession(ctx, sessionID)
			if err != nil {
				return nil, domain.MutationOK{}, fmt.Errorf("failed to create session: %w", err)
			}

			session, err := ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				return nil, domain.MutationOK{}, fmt.Errorf("failed to get session: %w", err)
			}

			var token ratchetdomain.TokenValue
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_link_cells")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_link_cells"), token)
			if err != nil {
				return nil, domain.MutationOK{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, in)
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_link_cells"))
			if err != nil {
				if ratchetWrapper.log != nil {
					ratchetWrapper.log.WarnContext(ctx, "failed to issue ratchet token", "error", err)
				}
			}

			session, err = ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				if ratchetWrapper.log != nil {
					ratchetWrapper.log.WarnContext(ctx, "failed to get session after token issuance", "error", err)
				}
			} else {
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_link_cells"))
				if updateErr := ratchetWrapper.sessionStore.Update(ctx, session); updateErr != nil {
					if ratchetWrapper.log != nil {
						ratchetWrapper.log.WarnContext(ctx, "failed to update session with tool call", "error", updateErr)
					}
				}
			}

			return result, resp, nil
		}
		handler = wrappedHandler
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_link_cells",
		Description: "Create or replace an edge between two axial cells Tx.LinkCells → Tx.PutEdge (relation_type, weight, provenance timestamps set to now). Direction matters for graph traversals AscendEdgesFrom(from).",
	}, handler)
}

func handleLinkCells(ctx context.Context, _ *mcp.CallToolRequest, in *linkInput, svc primary.FacetEdge, log *slog.Logger) (*mcp.CallToolResult, domain.MutationOK, error) {
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
}
