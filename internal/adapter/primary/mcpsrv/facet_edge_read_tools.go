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

type getFacetInput struct {
	Q       int  `json:"q"`
	R       int  `json:"r"`
	FacetID byte `json:"facet_id" jsonschema:"0..5 facet slot"`
}

type listFacetsInput struct {
	Q int `json:"q"`
	R int `json:"r"`
}

type getEdgeInput struct {
	FromQ        int    `json:"from_q"`
	FromR        int    `json:"from_r"`
	ToQ          int    `json:"to_q"`
	ToR          int    `json:"to_r"`
	RelationType string `json:"relation_type" jsonschema:"exact edge key label"`
}

type listEdgesFromInput struct {
	FromQ    int `json:"from_q"`
	FromR    int `json:"from_r"`
	MaxEdges int `json:"max_edges,omitempty" jsonschema:"optional cap default 50 max 200; truncated flag if more existed"`
}

// RegisterFacetEdgeBrowseTools registers read-only facet and edge lookup tools — View-only, no mutations.
func RegisterFacetEdgeBrowseTools(server *mcp.Server, svc primary.FacetEdgeBrowse, log *slog.Logger, budget *RetrievalBudgetTracker, ratchetWrapper *RatchetWrapper) {
	registerGetFacet(server, svc, log, budget, ratchetWrapper)
	registerListFacets(server, svc, log, budget, ratchetWrapper)
	registerGetEdge(server, svc, log, budget, ratchetWrapper)
	registerListEdgesFrom(server, svc, log, budget, ratchetWrapper)
}

func registerGetFacet(server *mcp.Server, svc primary.FacetEdgeBrowse, log *slog.Logger, budget *RetrievalBudgetTracker, ratchetWrapper *RatchetWrapper) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, in getFacetInput) (*mcp.CallToolResult, domain.GetFacetResponse, error) {
		return handleGetFacet(ctx, req, &in, svc, log, budget)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, in getFacetInput) (*mcp.CallToolResult, domain.GetFacetResponse, error) {
			sessionID := ratchetWrapper.DeriveSessionID(ctx)

			// Use ratchetSvc.CreateSession for session_created events
			_, err := ratchetWrapper.ratchetSvc.CreateSession(ctx, sessionID)
			if err != nil {
				return nil, domain.GetFacetResponse{}, fmt.Errorf("failed to create session: %w", err)
			}

			session, err := ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				return nil, domain.GetFacetResponse{}, fmt.Errorf("failed to get session: %w", err)
			}

			var token ratchetdomain.TokenValue
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_get_facet")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_get_facet"), token)
			if err != nil {
				return nil, domain.GetFacetResponse{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, in)
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_get_facet"))
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
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_get_facet"))
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
		Name:        "mosaic_hexxla_get_facet",
		Description: "Read one facet slot at (q,r) via Tx.GetFacet — read-only snapshot. Pair with mosaic_hexxla_put_facet verification before overwriting.",
	}, handler)
}

func handleGetFacet(ctx context.Context, req *mcp.CallToolRequest, in *getFacetInput, svc primary.FacetEdgeBrowse, log *slog.Logger, budget *RetrievalBudgetTracker) (*mcp.CallToolResult, domain.GetFacetResponse, error) {
	if log != nil {
		log.DebugContext(ctx, "mosaic_hexxla_get_facet")
	}
	out, err := RunBudgetedRead(budget, req, func() (domain.GetFacetResponse, error) {
		return svc.GetFacet(ctx, domain.AxialCoord{Q: in.Q, R: in.R}, in.FacetID)
	})
	return nil, out, err
}

func registerListFacets(server *mcp.Server, svc primary.FacetEdgeBrowse, log *slog.Logger, budget *RetrievalBudgetTracker, ratchetWrapper *RatchetWrapper) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, in listFacetsInput) (*mcp.CallToolResult, domain.ListFacetsForCellResponse, error) {
		return handleListFacets(ctx, req, &in, svc, log, budget)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, in listFacetsInput) (*mcp.CallToolResult, domain.ListFacetsForCellResponse, error) {
			sessionID := ratchetWrapper.DeriveSessionID(ctx)

			// Use ratchetSvc.CreateSession for session_created events
			_, err := ratchetWrapper.ratchetSvc.CreateSession(ctx, sessionID)
			if err != nil {
				return nil, domain.ListFacetsForCellResponse{}, fmt.Errorf("failed to create session: %w", err)
			}

			session, err := ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				return nil, domain.ListFacetsForCellResponse{}, fmt.Errorf("failed to get session: %w", err)
			}

			var token ratchetdomain.TokenValue
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_list_facets")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_list_facets"), token)
			if err != nil {
				return nil, domain.ListFacetsForCellResponse{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, in)
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_list_facets"))
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
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_list_facets"))
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
		Name:        "mosaic_hexxla_list_facets",
		Description: "List all visible facets at (q,r) via Tx.AscendFacetsForCell — read-only snapshot (slots returned in engine order).",
	}, handler)
}

func handleListFacets(ctx context.Context, req *mcp.CallToolRequest, in *listFacetsInput, svc primary.FacetEdgeBrowse, log *slog.Logger, budget *RetrievalBudgetTracker) (*mcp.CallToolResult, domain.ListFacetsForCellResponse, error) {
	if log != nil {
		log.DebugContext(ctx, "mosaic_hexxla_list_facets")
	}
	out, err := RunBudgetedRead(budget, req, func() (domain.ListFacetsForCellResponse, error) {
		return svc.ListFacetsForCell(ctx, domain.AxialCoord{Q: in.Q, R: in.R})
	})
	return nil, out, err
}

func registerGetEdge(server *mcp.Server, svc primary.FacetEdgeBrowse, log *slog.Logger, budget *RetrievalBudgetTracker, ratchetWrapper *RatchetWrapper) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, in getEdgeInput) (*mcp.CallToolResult, domain.GetEdgeResponse, error) {
		return handleGetEdge(ctx, req, &in, svc, log, budget)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, in getEdgeInput) (*mcp.CallToolResult, domain.GetEdgeResponse, error) {
			sessionID := ratchetWrapper.DeriveSessionID(ctx)

			// Use ratchetSvc.CreateSession for session_created events
			_, err := ratchetWrapper.ratchetSvc.CreateSession(ctx, sessionID)
			if err != nil {
				return nil, domain.GetEdgeResponse{}, fmt.Errorf("failed to create session: %w", err)
			}

			session, err := ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				return nil, domain.GetEdgeResponse{}, fmt.Errorf("failed to get session: %w", err)
			}

			var token ratchetdomain.TokenValue
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_get_edge")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_get_edge"), token)
			if err != nil {
				return nil, domain.GetEdgeResponse{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, in)
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_get_edge"))
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
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_get_edge"))
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
		Name:        "mosaic_hexxla_get_edge",
		Description: "Read one directed edge (from_cell → to_cell, relation_type) via Tx.GetEdge — read-only. Direction matches mosaic_hexxla_link_cells / AscendEdgesFrom(from).",
	}, handler)
}

func handleGetEdge(ctx context.Context, req *mcp.CallToolRequest, in *getEdgeInput, svc primary.FacetEdgeBrowse, log *slog.Logger, budget *RetrievalBudgetTracker) (*mcp.CallToolResult, domain.GetEdgeResponse, error) {
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
}

func registerListEdgesFrom(server *mcp.Server, svc primary.FacetEdgeBrowse, log *slog.Logger, budget *RetrievalBudgetTracker, ratchetWrapper *RatchetWrapper) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, in listEdgesFromInput) (*mcp.CallToolResult, domain.ListEdgesFromResponse, error) {
		return handleListEdgesFrom(ctx, req, &in, svc, log, budget)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, in listEdgesFromInput) (*mcp.CallToolResult, domain.ListEdgesFromResponse, error) {
			sessionID := ratchetWrapper.DeriveSessionID(ctx)

			// Use ratchetSvc.CreateSession for session_created events
			_, err := ratchetWrapper.ratchetSvc.CreateSession(ctx, sessionID)
			if err != nil {
				return nil, domain.ListEdgesFromResponse{}, fmt.Errorf("failed to create session: %w", err)
			}

			session, err := ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				return nil, domain.ListEdgesFromResponse{}, fmt.Errorf("failed to get session: %w", err)
			}

			var token ratchetdomain.TokenValue
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_list_edges_from")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_list_edges_from"), token)
			if err != nil {
				return nil, domain.ListEdgesFromResponse{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, in)
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_list_edges_from"))
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
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_list_edges_from"))
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
		Name:        "mosaic_hexxla_list_edges_from",
		Description: "List outbound edges whose from-cell equals (from_q,from_r) via Tx.AscendEdgesFrom — read-only. Bounded by max_edges (default 50, cap 200); truncated=true when scan stopped early at the cap.",
	}, handler)
}

func handleListEdgesFrom(ctx context.Context, req *mcp.CallToolRequest, in *listEdgesFromInput, svc primary.FacetEdgeBrowse, log *slog.Logger, budget *RetrievalBudgetTracker) (*mcp.CallToolResult, domain.ListEdgesFromResponse, error) {
	if log != nil {
		log.DebugContext(ctx, "mosaic_hexxla_list_edges_from")
	}
	out, err := RunBudgetedRead(budget, req, func() (domain.ListEdgesFromResponse, error) {
		return svc.ListEdgesFrom(ctx, domain.AxialCoord{Q: in.FromQ, R: in.FromR}, in.MaxEdges)
	})
	return nil, out, err
}
