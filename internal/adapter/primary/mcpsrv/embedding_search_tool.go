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

type embeddingSearchInput struct {
	Query      string  `json:"query" jsonschema:"natural language query to embed and match against stored embeddings"`
	MaxResults int     `json:"max_results,omitempty" jsonschema:"max hits (default 10, cap 50)"`
	MinScore   float64 `json:"min_score,omitempty" jsonschema:"minimum similarity score (0 = no filter)"`
}

// RegisterEmbeddingSearchTool registers mosaic_hexxla_search_embedding (semantic retrieval via Ollama + HexxlaDB ANN).
func RegisterEmbeddingSearchTool(server *mcp.Server, svc primary.EmbeddingSearch, log *slog.Logger, budget *RetrievalBudgetTracker, ratchetWrapper *RatchetWrapper) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, in embeddingSearchInput) (*mcp.CallToolResult, domain.EmbeddingSearchResponse, error) {
		return handleEmbeddingSearch(ctx, req, &in, svc, log, budget)
	}

	// Wrap with ratchet if provided
	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, in embeddingSearchInput) (*mcp.CallToolResult, domain.EmbeddingSearchResponse, error) {
			sessionID := ratchetWrapper.DeriveSessionID(ctx)

			// Get or create session
			session, err := ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				session = ratchetdomain.NewSession(sessionID)
				if createErr := ratchetWrapper.sessionStore.Create(ctx, session); createErr != nil {
					if ratchetWrapper.log != nil {
						ratchetWrapper.log.WarnContext(ctx, "failed to create session", "error", createErr)
					}
				}
			}

			// Get stored token for this tool from session
			var token ratchetdomain.TokenValue
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_search_embedding")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			// Validate tool call
			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_search_embedding"), token)
			if err != nil {
				return nil, domain.EmbeddingSearchResponse{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			// Execute the handler
			result, resp, err := originalHandler(ctx, req, in)
			if err != nil {
				return result, resp, err
			}

			// Issue token after successful execution
			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_search_embedding"))
			if err != nil {
				if ratchetWrapper.log != nil {
					ratchetWrapper.log.WarnContext(ctx, "failed to issue ratchet token", "error", err)
				}
			}

			// Re-fetch session after IssueToken
			session, err = ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				if ratchetWrapper.log != nil {
					ratchetWrapper.log.WarnContext(ctx, "failed to get session after token issuance", "error", err)
				}
			} else {
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_search_embedding"))
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
		Name: "mosaic_hexxla_search_embedding",
		Description: "Semantic retrieval: embed the query via Ollama (MOSAIC_EMBED_MODEL) and run HexxlaDB SearchByEmbedding ANN — returns top-K similar cells (coords, scores, text, tags). " +
			"Results include tags that can be used for category filtering (e.g. preference, fact, opinion, idea, code, signal, task, project). " +
			"Use as a first step to get seed coordinates; then call mosaic_hexxla_load_context_pack with 1-3 of those {q,r} to expand hex-neighbourhood context. " +
			"This tool alone is NOT full conversational context: neighbours, seams, and supersession may be missing. response JSON includes retrieval_hint.",
	}, handler)
}

func handleEmbeddingSearch(ctx context.Context, req *mcp.CallToolRequest, in *embeddingSearchInput, svc primary.EmbeddingSearch, log *slog.Logger, budget *RetrievalBudgetTracker) (*mcp.CallToolResult, domain.EmbeddingSearchResponse, error) {
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
}
