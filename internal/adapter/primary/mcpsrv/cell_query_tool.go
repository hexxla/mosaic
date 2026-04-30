package mcpsrv

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
)

// RegisterCellQueryTool registers mosaic_hexxla_query_cells (Hexxla Tx.QueryCells: tags, time, spatial, sort, optional explain).
func RegisterCellQueryTool(server *mcp.Server, svc primary.CellRetrieval, log *slog.Logger, budget *RetrievalBudgetTracker) {
	type cellQueryInput struct {
		Query          string   `json:"query,omitempty" jsonschema:"substring match on content, tags, or source_id"`
		RequireTags    []string `json:"require_tags,omitempty" jsonschema:"all of these tags (AND)"`
		AnyTags        []string `json:"any_tags,omitempty" jsonschema:"at least one tag (OR)"`
		ExcludeTags    []string `json:"exclude_tags,omitempty" jsonschema:"exclude if any of these tags present"`
		SourceID       string   `json:"source_id,omitempty"`
		MinConfidence  float64  `json:"min_confidence,omitempty"`
		MaxConfidence  float64  `json:"max_confidence,omitempty"`
		AfterRFC3339   string   `json:"after_rfc3339,omitempty" jsonschema:"ValidFrom lower bound (RFC3339)"`
		BeforeRFC3339  string   `json:"before_rfc3339,omitempty" jsonschema:"ValidFrom upper bound (RFC3339)"`
		CenterQ        *int     `json:"center_q,omitempty"`
		CenterR        *int     `json:"center_r,omitempty"`
		Radius         int      `json:"radius,omitempty" jsonschema:"hex rings from center; 0 disables spatial filter"`
		MaxResults     int      `json:"max_results,omitempty" jsonschema:"default 20, max 100"`
		MaxScanRows    int      `json:"max_scan_rows,omitempty" jsonschema:"cap index rows scanned (safety)"`
		SortBy         string   `json:"sort_by,omitempty" jsonschema:"score|confidence|recency|coord"`
		Explain        bool     `json:"explain,omitempty" jsonschema:"populate per-hit explanation text"`
		EmbedQueryText string   `json:"embed_query_text,omitempty" jsonschema:"when set, Ollama embeds this text and Hexxla uses ANN+filters hybrid QueryCells"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "mosaic_hexxla_query_cells",
		Description: "Indexed cell query (HexxlaDB QueryCells): tags, source, time window, spatial radius, sort, explain. " +
			"Use require_tags (e.g. preference) for structured slices instead of vague query-only search when fetching tagged memories. " +
			"Optional embed_query_text: Ollama embeds then ANN-accelerated candidate selection with same predicates (hybrid retrieval). Requires DB embeddings + MOSAIC_OLLAMA for hybrid. " +
			"Returns ranked hits only — mosaic_hexxla_load_context_pack with hit coords for neighbourhood context (retrieval_hint).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in cellQueryInput) (*mcp.CallToolResult, domain.CellHitsResponse, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_query_cells invoked")
		}
		after, err := parseOptionalRFC3339(in.AfterRFC3339)
		if err != nil {
			return nil, domain.CellHitsResponse{}, fmt.Errorf("after_rfc3339: %w", err)
		}
		before, err := parseOptionalRFC3339(in.BeforeRFC3339)
		if err != nil {
			return nil, domain.CellHitsResponse{}, fmt.Errorf("before_rfc3339: %w", err)
		}
		center, err := parseAxialCenter(in.CenterQ, in.CenterR)
		if err != nil {
			return nil, domain.CellHitsResponse{}, err
		}
		sortBy, err := parseCellQuerySort(in.SortBy)
		if err != nil {
			return nil, domain.CellHitsResponse{}, err
		}
		cmd := &domain.CellQueryCommand{
			Query:          strings.TrimSpace(in.Query),
			RequireTags:    append([]string(nil), in.RequireTags...),
			AnyTags:        append([]string(nil), in.AnyTags...),
			ExcludeTags:    append([]string(nil), in.ExcludeTags...),
			SourceID:       strings.TrimSpace(in.SourceID),
			MinConfidence:  in.MinConfidence,
			MaxConfidence:  in.MaxConfidence,
			After:          after,
			Before:         before,
			Center:         center,
			Radius:         in.Radius,
			MaxResults:     in.MaxResults,
			MaxScanRows:    in.MaxScanRows,
			SortBy:         sortBy,
			Explain:        in.Explain,
			EmbedQueryText: strings.TrimSpace(in.EmbedQueryText),
		}
		out, err := RunBudgetedRead(budget, req, func() (domain.CellHitsResponse, error) {
			hits, err2 := svc.QueryCells(ctx, cmd)
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
