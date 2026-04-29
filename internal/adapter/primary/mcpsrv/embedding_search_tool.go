package mcpsrv

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/ports/primary"
)

// RegisterEmbeddingSearchTool registers mosaic_hexxla_search_embedding (Ollama query embed + Hexxla SearchByEmbedding).
func RegisterEmbeddingSearchTool(server *mcp.Server, svc primary.EmbeddingSearch, log *slog.Logger, budget *RetrievalBudgetTracker) {
	type embeddingSearchInput struct {
		Query      string  `json:"query" jsonschema:"natural language query to embed and match against stored embeddings"`
		MaxResults int     `json:"max_results,omitempty" jsonschema:"max hits (default 10, cap 50)"`
		MinScore   float64 `json:"min_score,omitempty" jsonschema:"minimum similarity score (0 = no filter)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_search_embedding",
		Description: "Semantic retrieval: embed the query via Ollama (MOSAIC_EMBED_MODEL) and run HexxlaDB SearchByEmbedding ANN — returns top-K similar cells (coords, scores, text, tags). This is NOT full conversational context: neighbours, seams, and supersession may be missing. If the answer is incomplete, call mosaic_hexxla_load_context_pack with coords from matches as seeds (response JSON includes retrieval_hint).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in embeddingSearchInput) (*mcp.CallToolResult, domain.EmbeddingSearchResponse, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_search_embedding invoked")
		}
		out, err := RunBudgetedRead(budget, req, func() (domain.EmbeddingSearchResponse, error) {
			return svc.Search(ctx, domain.EmbeddingSearchQuery{
				Text:       in.Query,
				MaxResults: in.MaxResults,
				MinScore:   in.MinScore,
			})
		})
		if err != nil {
			return nil, domain.EmbeddingSearchResponse{}, err
		}
		return nil, out, nil
	})
}
