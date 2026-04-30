package mcpsrv

import (
	"context"
	"log/slog"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
)

// RegisterCellSearchTool registers mosaic_hexxla_search_cells (Hexxla Tx.SearchCells: lexical relevance + filters).
func RegisterCellSearchTool(server *mcp.Server, svc primary.CellRetrieval, log *slog.Logger, budget *RetrievalBudgetTracker) {
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

	mcp.AddTool(server, &mcp.Tool{
		Name: "mosaic_hexxla_search_cells",
		Description: "Lexical relevance search (HexxlaDB SearchCells): scored substring/tag/source matches; optional scan radius; optional embed_query_text for ANN-accelerated hybrid retrieval. " +
			"Substring match can return zero hits if that phrase is not stored literally — try mosaic_hexxla_search_embedding for semantic discovery. " +
			"Requires DB embeddings + MOSAIC_OLLAMA when embed_query_text set. Top hits only — mosaic_hexxla_load_context_pack for lattice context (retrieval_hint).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in cellSearchInput) (*mcp.CallToolResult, domain.CellHitsResponse, error) {
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
	})
}
