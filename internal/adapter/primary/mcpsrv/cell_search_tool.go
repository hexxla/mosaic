package mcpsrv

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hexxla/mosaic/internal/core/domain"

	"github.com/hexxla/mosaic/internal/core/ports/primary"
)

// RegisterCellSearchTool exposes one global-by-default discovery contract.
func RegisterCellSearchTool(server *mcp.Server, svc primary.CellRetrieval, log *slog.Logger, budget *RetrievalBudgetTracker) error {
	type input struct {
		Snapshot                *domain.SnapshotSelector `json:"snapshot,omitempty" jsonschema:"optional retained snapshot: exactly one of seq, tag or RFC3339 at; historical edges/vectors unavailable"`
		Embedding               []float32                `json:"embedding,omitempty" jsonschema:"optional query vector instead of Ollama query text; requires embedding_model"`
		EmbeddingModel          string                   `json:"embedding_model,omitempty" jsonschema:"caller-declared model identity for raw query vector"`
		EfSearch                int                      `json:"ef_search,omitzero" jsonschema:"semantic ANN breadth, maximum 10000"`
		EmbeddingCandidateLimit int                      `json:"embedding_candidate_limit,omitzero" jsonschema:"hybrid candidate ceiling, maximum 10000; incompatible with max_scan_rows"`
		Query                   string                   `json:"query,omitempty" jsonschema:"lexical text or semantic/hybrid embedding cue"`
		Mode                    string                   `json:"mode,omitempty" jsonschema:"lexical (default), semantic (pure vector), or hybrid (vector plus structured/optional lexical filters)"`
		LexicalQuery            string                   `json:"lexical_query,omitempty" jsonschema:"hybrid-only independent literal filter; omitted means no literal predicate"`
		RequireTags             []string                 `json:"require_tags,omitempty"`
		AnyTags                 []string                 `json:"any_tags,omitempty"`
		ExcludeTags             []string                 `json:"exclude_tags,omitempty"`
		SourceID                string                   `json:"source_id,omitempty"`
		MinConfidence           float64                  `json:"min_confidence,omitempty"`
		MaxConfidence           float64                  `json:"max_confidence,omitempty"`
		After                   string                   `json:"after_rfc3339,omitempty" jsonschema:"ValidFrom lower bound"`
		Before                  string                   `json:"before_rfc3339,omitempty" jsonschema:"ValidFrom upper bound"`
		CenterQ                 *int                     `json:"center_q,omitempty"`
		CenterR                 *int                     `json:"center_r,omitempty"`
		Radius                  int                      `json:"radius,omitempty" jsonschema:"explicit spatial scope; zero searches globally"`
		MaxResults              int                      `json:"max_results,omitempty" jsonschema:"default 20, maximum 100"`
		MaxScanRows             int                      `json:"max_scan_rows,omitempty" jsonschema:"lexical/hybrid scan cap up to 20000; hybrid requires a bounded filter"`
		SortBy                  string                   `json:"sort_by,omitempty" jsonschema:"score, confidence, recency, or coord; semantic uses score"`
		Explain                 bool                     `json:"explain,omitempty"`
		MinScore                float64                  `json:"min_score,omitempty" jsonschema:"semantic-only minimum vector score"`
	}
	return addTool(server, &mcp.Tool{Name: "mosaic_hexxla_search_cells", Description: "Discover cells globally unless an explicit radius is supplied. Lexical mode needs no model. Semantic mode preserves pure-vector scores without filters; use hybrid mode for filtered semantic retrieval and optional lexical_query. Scores are mode-specific. Inspect scan/candidate-limit flags; load_context_pack expands returned coordinates."}, func(ctx context.Context, req *mcp.CallToolRequest, in input) (*mcp.CallToolResult, domain.CellHitsResponse, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_search_cells invoked")
		}
		center, err := parseAxialCenter(in.CenterQ, in.CenterR)
		if err != nil {
			return nil, domain.CellHitsResponse{}, err
		}
		after, err := parseOptionalRFC3339(in.After)
		if err != nil {
			return nil, domain.CellHitsResponse{}, fmt.Errorf("after_rfc3339: %w", err)
		}
		before, err := parseOptionalRFC3339(in.Before)
		if err != nil {
			return nil, domain.CellHitsResponse{}, fmt.Errorf("before_rfc3339: %w", err)
		}
		sort, err := parseCellQuerySort(in.SortBy)
		if err != nil {
			return nil, domain.CellHitsResponse{}, err
		}
		q := &domain.CellSearchCommand{Snapshot: in.Snapshot, Embedding: in.Embedding, EmbeddingModel: in.EmbeddingModel, EfSearch: in.EfSearch, EmbeddingCandidateLimit: in.EmbeddingCandidateLimit, Query: strings.TrimSpace(in.Query), Mode: domain.SearchMode(strings.TrimSpace(in.Mode)), LexicalQuery: in.LexicalQuery, RequireTags: slices.Clone(in.RequireTags), AnyTags: slices.Clone(in.AnyTags), ExcludeTags: slices.Clone(in.ExcludeTags), SourceID: strings.TrimSpace(in.SourceID), MinConfidence: in.MinConfidence, MaxConfidence: in.MaxConfidence, After: after, Before: before, Center: center, Radius: in.Radius, MaxResults: in.MaxResults, MaxScanRows: in.MaxScanRows, SortBy: sort, Explain: in.Explain, MinScore: in.MinScore}
		out, err := budget.RunBudgetedRead(req, func() (domain.CellHitsResponse, error) { return svc.SearchCells(ctx, q) })
		return nil, out, err
	})
}

// RegisterGetCellTool exposes exact lookup for known identities.
func RegisterGetCellTool(server *mcp.Server, svc primary.CellRetrieval, log *slog.Logger, budget *RetrievalBudgetTracker) error {
	type input struct {
		Q int `json:"q"`
		R int `json:"r"`
	}
	return addTool(server, &mcp.Tool{Name: "mosaic_hexxla_get_cell", Description: "Read one live cell at exact axial q,r. Returns found:false for a missing or deleted identity. No search radius, model, or ranking is involved."}, func(ctx context.Context, req *mcp.CallToolRequest, in input) (*mcp.CallToolResult, domain.CellLookupResponse, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_get_cell invoked")
		}
		out, err := budget.RunBudgetedRead(req, func() (domain.CellLookupResponse, error) {
			return svc.GetCell(ctx, domain.AxialCoord{Q: in.Q, R: in.R})
		})
		return nil, out, err
	})
}
