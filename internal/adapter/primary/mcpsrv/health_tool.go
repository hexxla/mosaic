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

// RegisterHealthTool registers the mosaic_hexxla_health MCP tool, which delegates to [primary.Health].
func RegisterHealthTool(server *mcp.Server, health primary.Health, log *slog.Logger, budget *RetrievalBudgetTracker, ratchetWrapper *RatchetWrapper) {
	type healthInput struct{} // No parameters.

	handler := func(ctx context.Context, req *mcp.CallToolRequest, _ healthInput) (*mcp.CallToolResult, domain.HealthSummary, error) {
		return handleHealth(ctx, req, health, log, budget)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, _ healthInput) (*mcp.CallToolResult, domain.HealthSummary, error) {
			sessionID := ratchetWrapper.DeriveSessionID(ctx)

			// Use ratchetSvc.CreateSession for session_created events
			_, err := ratchetWrapper.ratchetSvc.CreateSession(ctx, sessionID)
			if err != nil {
				return nil, domain.HealthSummary{}, fmt.Errorf("failed to create session: %w", err)
			}

			session, err := ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				return nil, domain.HealthSummary{}, fmt.Errorf("failed to get session: %w", err)
			}

			var token ratchetdomain.TokenValue
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_health")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_health"), token)
			if err != nil {
				return nil, domain.HealthSummary{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, healthInput{})
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_health"))
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
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_health"))
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
		Name:        "mosaic_hexxla_health",
		Description: "Run HexxlaDB HealthCheck (cells, seams, tag/source indexes, orphans, MVCC stats, warnings) plus database layout (page size, max value bytes, embedding dimension/metric), disk footprint (primary + WAL bytes, paths), effective mvcc_retain_commits_behind_head from Mosaic policy, and integrity_ok. Use before heavy retrieval when embedding setup or DB integrity is unknown; empty embedding hits may mean dimension mismatch or missing vectors.",
	}, handler)
}

func handleHealth(ctx context.Context, req *mcp.CallToolRequest, health primary.Health, log *slog.Logger, budget *RetrievalBudgetTracker) (*mcp.CallToolResult, domain.HealthSummary, error) {
	if log != nil {
		log.DebugContext(ctx, "mosaic_hexxla_health invoked")
	}
	out, err := RunBudgetedRead(budget, req, func() (domain.HealthSummary, error) {
		return health.Status(ctx)
	})
	if err != nil {
		return nil, domain.HealthSummary{}, err
	}
	return nil, out, nil
}
