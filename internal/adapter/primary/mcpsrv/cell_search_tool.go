package mcpsrv

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	ratchetdomain "github.com/hexxla/mcp-ratchet/pkg/ratchet/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
)

type cellSearchInput struct {
	Query          string   `json:"query,omitempty" jsonschema:"matches content, tags, source_id; empty matches all with filters"`
	RequireTags    []string `json:"require_tags,omitempty"`
	AnyTags        []string `json:"any_tags,omitempty"`
	MinConfidence  float64  `json:"min_confidence,omitempty"`
	MaxConfidence  float64  `json:"max_confidence,omitempty"`
	SourceID       string   `json:"source_id,omitempty"`
	CenterQ        *int     `json:"center_q,omitempty"`
	CenterR        *int     `json:"center_r,omitempty"`
	Radius         int      `json:"radius,omitempty" jsonschema:"when 0, engine scans up to max_scan_radius rings"`
	MaxResults     int      `json:"max_results,omitempty" jsonschema:"default 20, max 100"`
	MaxScanRadius  int      `json:"max_scan_radius,omitempty" jsonschema:"when radius is 0, scan radius from origin (engine default 32)"`
	EmbedQueryText string   `json:"embed_query_text,omitempty" jsonschema:"when set, Ollama embeds then hybrid ANN+lexical SearchCells"`
}

// RegisterCellSearchTool registers mosaic_hexxla_search_cells (HexxlaDB SearchCells: lexical/tag/source matches, optional ANN hybrid).
func RegisterCellSearchTool(server *mcp.Server, svc primary.CellRetrieval, log *slog.Logger, budget *RetrievalBudgetTracker, ratchetWrapper *RatchetWrapper) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, in cellSearchInput) (*mcp.CallToolResult, domain.CellHitsResponse, error) {
		return handleCellSearch(ctx, req, &in, svc, log, budget)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, in cellSearchInput) (*mcp.CallToolResult, domain.CellHitsResponse, error) {
			sessionID := ratchetWrapper.DeriveSessionID(ctx)

			// Use ratchetSvc.CreateSession for session_created events
			_, err := ratchetWrapper.ratchetSvc.CreateSession(ctx, sessionID)
			if err != nil {
				return nil, domain.CellHitsResponse{}, fmt.Errorf("failed to create session: %w", err)
			}

			session, err := ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				return nil, domain.CellHitsResponse{}, fmt.Errorf("failed to get session: %w", err)
			}

			var token ratchetdomain.TokenValue
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_search_cells")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_search_cells"), token)
			if err != nil {
				return nil, domain.CellHitsResponse{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, in)
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_search_cells"))
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
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_search_cells"))
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
		Name: "mosaic_hexxla_search_cells",
		Description: "Lexical relevance search (HexxlaDB SearchCells): scored substring/tag/source matches; optional scan radius; optional embed_query_text for ANN-accelerated hybrid retrieval. " +
			"Use tag filters (e.g. preference, fact, opinion, idea, code, signal, task, project) to narrow results by category. " +
			"Substring match can return zero hits if that phrase is not stored literally — try mosaic_hexxla_search_embedding for semantic discovery. " +
			"Requires DB embeddings + MOSAIC_OLLAMA when embed_query_text set. Top hits only — mosaic_hexxla_load_context_pack for lattice context (retrieval_hint).",
	}, handler)
}

func handleCellSearch(ctx context.Context, req *mcp.CallToolRequest, in *cellSearchInput, svc primary.CellRetrieval, log *slog.Logger, budget *RetrievalBudgetTracker) (*mcp.CallToolResult, domain.CellHitsResponse, error) {
	if log != nil {
		log.DebugContext(ctx, "mosaic_hexxla_search_cells invoked")
	}
	center, err := parseAxialCenter(in.CenterQ, in.CenterR)
	if err != nil {
		return nil, domain.CellHitsResponse{}, err
	}
	cmd := &domain.CellSearchCommand{
		Query:          strings.TrimSpace(in.Query),
		RequireTags:    append([]string(nil), in.RequireTags...),
		AnyTags:        append([]string(nil), in.AnyTags...),
		MinConfidence:  in.MinConfidence,
		MaxConfidence:  in.MaxConfidence,
		SourceID:       strings.TrimSpace(in.SourceID),
		Center:         center,
		Radius:         in.Radius,
		MaxResults:     in.MaxResults,
		MaxScanRadius:  in.MaxScanRadius,
		EmbedQueryText: strings.TrimSpace(in.EmbedQueryText),
	}
	out, err := RunBudgetedRead(budget, req, func() (domain.CellHitsResponse, error) {
		hits, err2 := svc.SearchCells(ctx, cmd)
		if err2 != nil {
			return domain.CellHitsResponse{}, err2
		}
		return domain.CellHitsResponse{
			Hits:          hits,
			RetrievalHint: domain.RetrievalHintLexicalOrANN,
		}, nil
	})
	if err != nil {
		return nil, domain.CellHitsResponse{}, err
	}
	return nil, out, nil
}
