package mcpsrv

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	ratchetdomain "github.com/hexxla/mcp-ratchet/pkg/ratchet/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
)

type seedCoord struct {
	Q int `json:"q" jsonschema:"axial q"`
	R int `json:"r" jsonschema:"axial r"`
}

type contextPackInput struct {
	Seeds []seedCoord `json:"seeds" jsonschema:"one or more lattice coords (often from embedding/query/search hits)"`
	// Optional defaults: max_ring 3, UTF-8 byte budget 4096 (see budget fields).
	MaxRing   int `json:"max_ring,omitempty" jsonschema:"hex rings from each seed (max 32)"`
	MaxCells  int `json:"max_cells,omitempty" jsonschema:"cap candidate pool before eviction (default engine 256)"`
	MaxTokens int `json:"max_tokens,omitempty" jsonschema:"legacy: explicit UTF-8 byte budget (ByteLenBudgeter); prefer max_budget_bytes"`

	OmitBudget bool `json:"omit_budget,omitempty" jsonschema:"sparse no tight cap: use maximum allowed UTF-8 byte budget (100000); exclusive with other budget fields"`

	BudgetTokensApprox  int     `json:"budget_tokens_approx,omitempty" jsonschema:"approximate LM tokens → bytes via bytes_per_approx_token (default 4); exclusive with omit_budget and explicit bytes"`
	BytesPerApproxToken float64 `json:"bytes_per_approx_token,omitempty" jsonschema:"bytes per approximate token for budget_tokens_approx (2–16, default 4)"`

	MaxBudgetBytes int `json:"max_budget_bytes,omitempty" jsonschema:"explicit UTF-8 byte ceiling; do not combine with budget_tokens_approx; may repeat max_tokens legacy for same value"`

	FilterSuperseded bool `json:"filter_superseded,omitempty"`
	IncludeSeams     bool `json:"include_seams,omitempty"`
	IncludeFacetText bool `json:"include_facet_text,omitempty"`
	Explain          bool `json:"explain,omitempty"`
}

func validateContextPackBudgetInput(in *contextPackInput) error {
	if in == nil {
		return errors.New("context pack budget: nil input")
	}
	if in.OmitBudget {
		if in.BudgetTokensApprox > 0 || in.MaxBudgetBytes > 0 || in.MaxTokens > 0 {
			return errors.New("context pack budget: omit_budget cannot be combined with other budget fields")
		}
		return nil
	}
	if in.BudgetTokensApprox > 0 {
		if in.MaxBudgetBytes > 0 || in.MaxTokens > 0 {
			return errors.New("context pack budget: budget_tokens_approx cannot be combined with max_budget_bytes or max_tokens")
		}
		return nil
	}
	if in.MaxBudgetBytes > 0 && in.MaxTokens > 0 && in.MaxBudgetBytes != in.MaxTokens {
		return errors.New("context pack budget: max_budget_bytes and max_tokens disagree; set one field or use the same value")
	}
	return nil
}

// RegisterContextPackTool registers mosaic_hexxla_load_context_pack (Hexxla Tx.LoadContextPackFrom).
func RegisterContextPackTool(server *mcp.Server, svc primary.ContextAssembly, log *slog.Logger, budget *RetrievalBudgetTracker, ratchetWrapper *RatchetWrapper) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, in contextPackInput) (*mcp.CallToolResult, domain.ContextPackResponse, error) {
		return handleContextPack(ctx, req, &in, svc, log, budget)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, in contextPackInput) (*mcp.CallToolResult, domain.ContextPackResponse, error) {
			sessionID := ratchetWrapper.DeriveSessionID(ctx)

			// Use ratchetSvc.CreateSession for session_created events
			_, err := ratchetWrapper.ratchetSvc.CreateSession(ctx, sessionID)
			if err != nil {
				return nil, domain.ContextPackResponse{}, fmt.Errorf("failed to create session: %w", err)
			}

			session, err := ratchetWrapper.sessionStore.Get(ctx, sessionID)
			if err != nil {
				return nil, domain.ContextPackResponse{}, fmt.Errorf("failed to get session: %w", err)
			}

			var token ratchetdomain.TokenValue
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_load_context_pack")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_load_context_pack"), token)
			if err != nil {
				return nil, domain.ContextPackResponse{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, in)
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_load_context_pack"))
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
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_load_context_pack"))
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
		Name: "mosaic_hexxla_load_context_pack",
		Description: "Expand hex-neighbourhood context from seed coordinates using HexxlaDB LoadContextPackFrom (ring walk + UTF-8 byte budget via ByteLenBudgeter, optional seams & supersession filtering). " +
			"Typical flow: mosaic_hexxla_search_embedding (or query/search cells) → use match coords as seeds here. " +
			"Budget: omit_budget (sparse cap), budget_tokens_approx (LM-token estimate × bytes/token), max_budget_bytes or legacy max_tokens (explicit bytes), else default ~4096 bytes. " +
			"Preview bytes with mosaic_hexxla_estimate_context_budget_bytes. " +
			"Start with a moderate max_ring and budget; increase if needed. " +
			"Embedding search alone returns similar cells only; this tool pulls adjacent lattice context (seams, neighbours) for prompts.",
	}, handler)
}

func handleContextPack(ctx context.Context, req *mcp.CallToolRequest, in *contextPackInput, svc primary.ContextAssembly, log *slog.Logger, budget *RetrievalBudgetTracker) (*mcp.CallToolResult, domain.ContextPackResponse, error) {
	if log != nil {
		log.DebugContext(ctx, "mosaic_hexxla_load_context_pack invoked")
	}
	if len(in.Seeds) == 0 {
		return nil, domain.ContextPackResponse{}, errors.New("seeds: at least one {q,r} required")
	}
	if err := validateContextPackBudgetInput(in); err != nil {
		return nil, domain.ContextPackResponse{}, err
	}
	seeds := make([]domain.AxialCoord, 0, len(in.Seeds))
	for _, s := range in.Seeds {
		seeds = append(seeds, domain.AxialCoord{Q: s.Q, R: s.R})
	}
	cmd := &domain.LoadContextPackCommand{
		Seeds:               seeds,
		MaxRing:             in.MaxRing,
		OmitBudget:          in.OmitBudget,
		BudgetTokensApprox:  in.BudgetTokensApprox,
		BytesPerApproxToken: in.BytesPerApproxToken,
		MaxBudgetBytes:      in.MaxBudgetBytes,
		MaxTokens:           in.MaxTokens,
		MaxCells:            in.MaxCells,
		FilterSuperseded:    in.FilterSuperseded,
		IncludeSeams:        in.IncludeSeams,
		IncludeFacetText:    in.IncludeFacetText,
		Explain:             in.Explain,
	}
	out, err := RunBudgetedRead(budget, req, func() (domain.ContextPackResponse, error) {
		return svc.LoadFromSeeds(ctx, cmd)
	})
	if err != nil {
		return nil, domain.ContextPackResponse{}, err
	}
	return nil, out, nil
}
