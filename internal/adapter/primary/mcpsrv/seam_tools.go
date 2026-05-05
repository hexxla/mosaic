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

type findSeamsInput struct {
	CenterQ        int  `json:"center_q" jsonschema:"axial q of search center"`
	CenterR        int  `json:"center_r" jsonschema:"axial r of search center"`
	Radius         *int `json:"radius,omitempty" jsonschema:"hex rings from center; omit for default 3, use 0 for center cell only"`
	UnresolvedOnly bool `json:"unresolved_only,omitempty" jsonschema:"only seams with empty resolution_status"`
}

type markConflictInput struct {
	Aq     int    `json:"aq" jsonschema:"cell A axial q"`
	Ar     int    `json:"ar" jsonschema:"cell A axial r"`
	Bq     int    `json:"bq" jsonschema:"cell B axial q"`
	Br     int    `json:"br" jsonschema:"cell B axial r"`
	Reason string `json:"reason,omitempty" jsonschema:"human-readable conflict reason"`
}

type markSupInput struct {
	SupersederQ int    `json:"superseder_q" jsonschema:"current-truth cell q"`
	SupersederR int    `json:"superseder_r" jsonschema:"current-truth cell r"`
	SupersededQ int    `json:"superseded_q" jsonschema:"superseded (stale) cell q"`
	SupersededR int    `json:"superseded_r" jsonschema:"superseded cell r"`
	Reason      string `json:"reason,omitempty"`
}

type resolveInput struct {
	SeamID           string `json:"seam_id" jsonschema:"26-char ULID from find_seams"`
	ResolutionStatus string `json:"resolution_status" jsonschema:"non-empty status label"`
	ResolutionNote   string `json:"resolution_note,omitempty"`
}

// RegisterSeamTools wires seam discovery and writes: find_seams, mark_conflict, mark_supersedes, resolve_seam.
func RegisterSeamTools(server *mcp.Server, svc primary.SeamLifecycle, log *slog.Logger, budget *RetrievalBudgetTracker, ratchetWrapper *RatchetWrapper) {
	registerFindSeamsTool(server, svc, log, budget, ratchetWrapper)
	registerMarkConflictTool(server, svc, log, ratchetWrapper)
	registerMarkSupersedesTool(server, svc, log, ratchetWrapper)
	registerResolveSeamTool(server, svc, log, ratchetWrapper)
}

func registerFindSeamsTool(server *mcp.Server, svc primary.SeamLifecycle, log *slog.Logger, budget *RetrievalBudgetTracker, ratchetWrapper *RatchetWrapper) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, in findSeamsInput) (*mcp.CallToolResult, domain.FindSeamsResponse, error) {
		return handleFindSeams(ctx, req, &in, svc, log, budget)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, in findSeamsInput) (*mcp.CallToolResult, domain.FindSeamsResponse, error) {
			sessionID := ratchetWrapper.DeriveSessionID(ctx)

			// Use ratchetSvc.CreateSession for session_created events
			_, err := ratchetWrapper.ratchetSvc.CreateSession(ctx, sessionID)
			if err != nil {
				return nil, domain.FindSeamsResponse{}, fmt.Errorf("failed to create session: %w", err)
			}

			session, err := ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				return nil, domain.FindSeamsResponse{}, fmt.Errorf("failed to get session: %w", err)
			}

			var token ratchetdomain.TokenValue
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_find_seams")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_find_seams"), token)
			if err != nil {
				return nil, domain.FindSeamsResponse{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, in)
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_find_seams"))
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
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_find_seams"))
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
		Name:        "mosaic_hexxla_find_seams",
		Description: "List seams HexxlaDB.FindSeams: endpoints within hex distance radius of (center_q, center_r). Set unresolved_only to focus on seams not yet ResolveSeam'd. Larger radii scan more neighbouring cells.",
	}, handler)
}

func handleFindSeams(ctx context.Context, req *mcp.CallToolRequest, in *findSeamsInput, svc primary.SeamLifecycle, log *slog.Logger, budget *RetrievalBudgetTracker) (*mcp.CallToolResult, domain.FindSeamsResponse, error) {
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
	return nil, out, err
}

func registerMarkConflictTool(server *mcp.Server, svc primary.SeamLifecycle, log *slog.Logger, ratchetWrapper *RatchetWrapper) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, in markConflictInput) (*mcp.CallToolResult, domain.MutationOK, error) {
		return handleMarkConflict(ctx, req, &in, svc, log)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, in markConflictInput) (*mcp.CallToolResult, domain.MutationOK, error) {
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
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_mark_conflict")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_mark_conflict"), token)
			if err != nil {
				return nil, domain.MutationOK{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, in)
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_mark_conflict"))
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
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_mark_conflict"))
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
		Name:        "mosaic_hexxla_mark_conflict",
		Description: "Create a contradiction seam Tx.MarkConflict between two axial cells — canonical endpoints, SeamType mark_conflict — for review or resolution workflows.",
	}, handler)
}

func handleMarkConflict(ctx context.Context, _ *mcp.CallToolRequest, in *markConflictInput, svc primary.SeamLifecycle, log *slog.Logger) (*mcp.CallToolResult, domain.MutationOK, error) {
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
}

func registerMarkSupersedesTool(server *mcp.Server, svc primary.SeamLifecycle, log *slog.Logger, ratchetWrapper *RatchetWrapper) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, in markSupInput) (*mcp.CallToolResult, domain.MutationOK, error) {
		return handleMarkSupersedes(ctx, req, &in, svc, log)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, in markSupInput) (*mcp.CallToolResult, domain.MutationOK, error) {
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
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_mark_supersedes")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_mark_supersedes"), token)
			if err != nil {
				return nil, domain.MutationOK{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, in)
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_mark_supersedes"))
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
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_mark_supersedes"))
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
		Name:        "mosaic_hexxla_mark_supersedes",
		Description: "Record supersession Tx.MarkSupersedes: superseder_* is current truth, superseded_* is stale. SeamType supersedes — used by mosaic_hexxla_load_context_pack when FilterSuperseded is enabled.",
	}, handler)
}

func handleMarkSupersedes(ctx context.Context, _ *mcp.CallToolRequest, in *markSupInput, svc primary.SeamLifecycle, log *slog.Logger) (*mcp.CallToolResult, domain.MutationOK, error) {
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
}

func registerResolveSeamTool(server *mcp.Server, svc primary.SeamLifecycle, log *slog.Logger, ratchetWrapper *RatchetWrapper) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, in resolveInput) (*mcp.CallToolResult, domain.MutationOK, error) {
		return handleResolveSeam(ctx, req, &in, svc, log)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, in resolveInput) (*mcp.CallToolResult, domain.MutationOK, error) {
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
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_resolve_seam")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_resolve_seam"), token)
			if err != nil {
				return nil, domain.MutationOK{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, in)
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_resolve_seam"))
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
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_resolve_seam"))
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
		Name:        "mosaic_hexxla_resolve_seam",
		Description: "Update resolution fields on an existing seam Tx.ResolveSeam. Fails with seam not found if the id is unknown.",
	}, handler)
}

func handleResolveSeam(ctx context.Context, _ *mcp.CallToolRequest, in *resolveInput, svc primary.SeamLifecycle, log *slog.Logger) (*mcp.CallToolResult, domain.MutationOK, error) {
	if log != nil {
		log.DebugContext(ctx, "mosaic_hexxla_resolve_seam invoked")
	}
	err := svc.ResolveSeam(ctx, in.SeamID, in.ResolutionStatus, in.ResolutionNote)
	if err != nil {
		return nil, domain.MutationOK{}, err
	}
	return nil, domain.MutationOK{OK: true}, nil
}
