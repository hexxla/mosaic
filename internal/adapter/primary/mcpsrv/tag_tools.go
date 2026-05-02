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

// RegisterTagBrowseTools registers read-only mosaic_hexxla_list_tags and mosaic_hexxla_tag_counts.
func RegisterTagBrowseTools(server *mcp.Server, svc primary.TagBrowse, log *slog.Logger, budget *RetrievalBudgetTracker, ratchetWrapper *RatchetWrapper) {
	// mosaic_hexxla_list_tags
	handler := func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, domain.DistinctTagsResult, error) {
		return handleListTags(ctx, req, svc, log, budget)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, domain.DistinctTagsResult, error) {
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
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_list_tags")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_list_tags"), token)
			if err != nil {
				return nil, domain.DistinctTagsResult{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, struct{}{})
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_list_tags"))
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
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_list_tags"))
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
		Name: "mosaic_hexxla_list_tags",
		Description: "List all distinct cell tags visible in the DB (sorted), same semantics as Hexxla Tx.ListExistingTopics — read-only. " +
			"Call before mosaic_hexxla_put_cell whenever tags might overlap existing vocabulary; reuse tags from this list when they fit. " +
			"When lexical search misses, align queries with vocabulary from here.",
	}, handler)

	// mosaic_hexxla_tag_counts
	handler2 := func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, domain.TagCountsResult, error) {
		return handleTagCounts(ctx, req, svc, log, budget)
	}

	if ratchetWrapper != nil {
		originalHandler2 := handler2
		wrappedHandler2 := func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, domain.TagCountsResult, error) {
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
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_tag_counts")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_tag_counts"), token)
			if err != nil {
				return nil, domain.TagCountsResult{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler2(ctx, req, struct{}{})
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_tag_counts"))
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
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_tag_counts"))
				if updateErr := ratchetWrapper.sessionStore.Update(ctx, session); updateErr != nil {
					if ratchetWrapper.log != nil {
						ratchetWrapper.log.WarnContext(ctx, "failed to update session with tool call", "error", updateErr)
					}
				}
			}

			return result, resp, nil
		}
		handler2 = wrappedHandler2
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "mosaic_hexxla_tag_counts",
		Description: "Per-tag visible cell counts (sorted high-to-low frequency), Hexxla Tx.TagCounts semantics — read-only. " +
			"Use with mosaic_hexxla_list_tags before put_cell to prefer high-frequency established tags over one-off spellings. Also narrows search/query filters.",
	}, handler2)
}

func handleListTags(ctx context.Context, req *mcp.CallToolRequest, svc primary.TagBrowse, log *slog.Logger, budget *RetrievalBudgetTracker) (*mcp.CallToolResult, domain.DistinctTagsResult, error) {
	if log != nil {
		log.Debug("mcp mosaic_hexxla_list_tags")
	}
	out, err := RunBudgetedRead(budget, req, func() (domain.DistinctTagsResult, error) {
		return svc.ListDistinctTags(ctx)
	})
	if err != nil {
		return nil, domain.DistinctTagsResult{}, err
	}
	return nil, out, nil
}

func handleTagCounts(ctx context.Context, req *mcp.CallToolRequest, svc primary.TagBrowse, log *slog.Logger, budget *RetrievalBudgetTracker) (*mcp.CallToolResult, domain.TagCountsResult, error) {
	if log != nil {
		log.Debug("mcp mosaic_hexxla_tag_counts")
	}
	out, err := RunBudgetedRead(budget, req, func() (domain.TagCountsResult, error) {
		return svc.TagCounts(ctx)
	})
	if err != nil {
		return nil, domain.TagCountsResult{}, err
	}
	return nil, out, nil
}
