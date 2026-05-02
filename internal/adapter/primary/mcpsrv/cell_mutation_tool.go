package mcpsrv

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	ratchetdomain "github.com/hexxla/mcp-ratchet/pkg/ratchet/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/config"
	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
)

type putCellInput struct {
	Q          int      `json:"q" jsonschema:"axial q"`
	R          int      `json:"r" jsonschema:"axial r"`
	RawContent string   `json:"raw_content" jsonschema:"cell body text"`
	Tags       []string `json:"tags,omitempty" jsonschema:"optional tag strings (merged with template tags)"`
	SourceID   string   `json:"source_id" jsonschema:"provenance source or session id"`
	Confidence float64  `json:"confidence" jsonschema:"0..1"`
	Kind       string   `json:"kind,omitempty" jsonschema:"fact (default) | user_message | assistant_response"`
}

type putEmbInput struct {
	Q      int       `json:"q" jsonschema:"axial q"`
	R      int       `json:"r" jsonschema:"axial r"`
	Text   string    `json:"text,omitempty" jsonschema:"natural language to embed via Ollama (mutually exclusive with vector)"`
	Vector []float32 `json:"vector,omitempty" jsonschema:"raw float32 embedding; length must match DB EmbeddingDimension"`
}

type deleteCellInput struct {
	Q int `json:"q" jsonschema:"axial q"`
	R int `json:"r" jsonschema:"axial r"`
}

// RegisterCellMutationTools registers write tools: mosaic_hexxla_put_cell, mosaic_hexxla_put_embedding, mosaic_hexxla_delete_cell.
func RegisterCellMutationTools(server *mcp.Server, svc primary.CellMutation, gates config.MosaicRuntimeConfig, log *slog.Logger, ratchetWrapper *RatchetWrapper) {
	registerPutCellTool(server, svc, gates, log, ratchetWrapper)
	registerPutEmbeddingTool(server, svc, log, ratchetWrapper)
	registerDeleteCellTool(server, svc, gates, log, ratchetWrapper)
}

func registerPutCellTool(server *mcp.Server, svc primary.CellMutation, gates config.MosaicRuntimeConfig, log *slog.Logger, ratchetWrapper *RatchetWrapper) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, in putCellInput) (*mcp.CallToolResult, domain.MutationOK, error) {
		return handlePutCell(ctx, req, &in, svc, gates, log)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, in putCellInput) (*mcp.CallToolResult, domain.MutationOK, error) {
			sessionID := ratchetWrapper.DeriveSessionID(ctx)

			session, err := ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				session = ratchetdomain.NewSession(sessionID)
				if createErr := ratchetWrapper.sessionStore.Create(ctx, session); createErr != nil {
					if ratchetWrapper.log != nil {
						ratchetWrapper.log.WarnContext(ctx, "failed to create session", "error", createErr)
					}
				}
			}

			var token ratchetdomain.TokenValue
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_put_cell")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_put_cell"), token)
			if err != nil {
				return nil, domain.MutationOK{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, in)
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_put_cell"))
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
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_put_cell"))
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
		Name: "mosaic_hexxla_put_cell",
		Description: "Write one cell at axial (q,r) via HexxlaDB PutCell (DB.Update). Before choosing tags, call mosaic_hexxla_list_tags (and mosaic_hexxla_tag_counts) when taxonomy is unknown — reuse existing tags when relevant instead of inventing near-duplicates. " +
			"**Tag Categories:** Use semantic tags to categorize content: " +
			"Content nature: preference (user likes/dislikes), fact (objective information), opinion (subjective views), idea (concepts/proposals), code (snippets/technical), signal (important/high-value), noise (low-value/irrelevant). " +
			"Actions: task (action items), decision (made choices), question (queries), answer (responses). " +
			"Work: project (work-related), bug (issues/errors), feature (capabilities), requirement (needs), goal (objectives), milestone (progress), deadline (time constraints), priority (importance), status (current state). " +
			"Communication: meeting (discussions), email (communications), document (files), note (general notes). " +
			"Personal: personal (private info), contact (people), location (geographical). " +
			"kind=fact uses a fact template; user_message / assistant_response match conversational_memory-style tags. source_id is required (provenance source for fact, session id for user/assistant templates).",
	}, handler)
}

func handlePutCell(ctx context.Context, _ *mcp.CallToolRequest, in *putCellInput, svc primary.CellMutation, gates config.MosaicRuntimeConfig, log *slog.Logger) (*mcp.CallToolResult, domain.MutationOK, error) {
	if log != nil {
		log.DebugContext(ctx, "mosaic_hexxla_put_cell invoked", "q", in.Q, "r", in.R)
	}
	kind := domain.CellPutKind(strings.TrimSpace(in.Kind))
	if kind == "" {
		kind = domain.CellPutKindFact
	}
	if err := gates.PutCellDenied(kind); err != nil {
		return nil, domain.MutationOK{}, fmt.Errorf("mosaic_hexxla_put_cell: %w", err)
	}
	err := svc.PutCell(ctx, &domain.PutCellCommand{
		Coord:      domain.AxialCoord{Q: in.Q, R: in.R},
		RawContent: in.RawContent,
		Tags:       append([]string(nil), in.Tags...),
		SourceID:   in.SourceID,
		Confidence: in.Confidence,
		Kind:       kind,
	})
	if err != nil {
		return nil, domain.MutationOK{}, err
	}
	return nil, domain.MutationOK{OK: true}, nil
}

func registerPutEmbeddingTool(server *mcp.Server, svc primary.CellMutation, log *slog.Logger, ratchetWrapper *RatchetWrapper) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, in putEmbInput) (*mcp.CallToolResult, domain.MutationOK, error) {
		return handlePutEmbedding(ctx, req, &in, svc, log)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, in putEmbInput) (*mcp.CallToolResult, domain.MutationOK, error) {
			sessionID := ratchetWrapper.DeriveSessionID(ctx)

			session, err := ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				session = ratchetdomain.NewSession(sessionID)
				if createErr := ratchetWrapper.sessionStore.Create(ctx, session); createErr != nil {
					if ratchetWrapper.log != nil {
						ratchetWrapper.log.WarnContext(ctx, "failed to create session", "error", createErr)
					}
				}
			}

			var token ratchetdomain.TokenValue
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_put_embedding")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_put_embedding"), token)
			if err != nil {
				return nil, domain.MutationOK{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, in)
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_put_embedding"))
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
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_put_embedding"))
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
		Name:        "mosaic_hexxla_put_embedding",
		Description: "Store an embedding vector at axial (q,r) via HexxlaDB Tx.PutEmbedding. Provide either text (embedded with MOSAIC_EMBED_MODEL) or vector (full float32[], length must match the database embedding dimension). Requires embeddings enabled on the DB (same dimension as MOSAIC_DB_PATH database).",
	}, handler)
}

func handlePutEmbedding(ctx context.Context, _ *mcp.CallToolRequest, in *putEmbInput, svc primary.CellMutation, log *slog.Logger) (*mcp.CallToolResult, domain.MutationOK, error) {
	if log != nil {
		log.DebugContext(ctx, "mosaic_hexxla_put_embedding invoked", "q", in.Q, "r", in.R)
	}
	err := svc.PutEmbedding(ctx, &domain.PutEmbeddingCommand{
		Coord:  domain.AxialCoord{Q: in.Q, R: in.R},
		Text:   in.Text,
		Vector: append([]float32(nil), in.Vector...),
	})
	if err != nil {
		return nil, domain.MutationOK{}, err
	}
	return nil, domain.MutationOK{OK: true}, nil
}

func registerDeleteCellTool(server *mcp.Server, svc primary.CellMutation, gates config.MosaicRuntimeConfig, log *slog.Logger, ratchetWrapper *RatchetWrapper) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, in deleteCellInput) (*mcp.CallToolResult, domain.DeleteCellMutationResult, error) {
		return handleDeleteCell(ctx, req, &in, svc, gates, log)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, in deleteCellInput) (*mcp.CallToolResult, domain.DeleteCellMutationResult, error) {
			sessionID := ratchetWrapper.DeriveSessionID(ctx)

			session, err := ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				session = ratchetdomain.NewSession(sessionID)
				if createErr := ratchetWrapper.sessionStore.Create(ctx, session); createErr != nil {
					if ratchetWrapper.log != nil {
						ratchetWrapper.log.WarnContext(ctx, "failed to create session", "error", createErr)
					}
				}
			}

			var token ratchetdomain.TokenValue
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_delete_cell")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_delete_cell"), token)
			if err != nil {
				return nil, domain.DeleteCellMutationResult{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, in)
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_delete_cell"))
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
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_delete_cell"))
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
		Name: "mosaic_hexxla_delete_cell",
		Description: "Delete the cell at axial (q,r) via HexxlaDB Tx.DeleteCell (MVCC tombstone semantics on v2 databases). " +
			"Successful response includes cell_removed: true only when a visible cell existed and was removed; " +
			"false means the coordinate had no live cell (already deleted, wrong coords, or never written) — not an error.",
	}, handler)
}

func handleDeleteCell(ctx context.Context, _ *mcp.CallToolRequest, in *deleteCellInput, svc primary.CellMutation, gates config.MosaicRuntimeConfig, log *slog.Logger) (*mcp.CallToolResult, domain.DeleteCellMutationResult, error) {
	if log != nil {
		log.DebugContext(ctx, "mosaic_hexxla_delete_cell invoked", "q", in.Q, "r", in.R)
	}
	if err := gates.DeleteCellDenied(); err != nil {
		return nil, domain.DeleteCellMutationResult{}, fmt.Errorf("mosaic_hexxla_delete_cell: %w", err)
	}
	removed, err := svc.DeleteCell(ctx, &domain.DeleteCellCommand{
		Coord: domain.AxialCoord{Q: in.Q, R: in.R},
	})
	if err != nil {
		return nil, domain.DeleteCellMutationResult{}, err
	}
	return nil, domain.DeleteCellMutationResult{OK: true, CellRemoved: removed}, nil
}
