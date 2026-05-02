package mcpsrv

import (
	"context"
	"fmt"
	"log/slog"

	ratchetdomain "github.com/hexxla/mcp-ratchet/pkg/ratchet/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/config"
)

// MosaicConfigPolicyResponse is the MCP JSON payload for mosaic_hexxla_get_persistence_policy.
type MosaicConfigPolicyResponse struct {
	Version         int                       `json:"version"`
	Retention       mosaicRetentionPolicyJSON `json:"retention"`
	AllowDeleteCell bool                      `json:"allow_delete_cell"`
	ConfigFile      string                    `json:"config_file,omitempty"`
	IsDefaultConfig bool                      `json:"is_default"`
}

// mosaicRetentionPolicyJSON mirrors config.RetentionPolicy with enforcement as a boolean for MCP clients.
type mosaicRetentionPolicyJSON struct {
	Version     int    `json:"version"`
	CaptureMode string `json:"capture_mode"`
	Enforcement bool   `json:"enforcement"`
	Notes       string `json:"notes,omitempty"`
}

func newMosaicRetentionPolicyJSON(r config.RetentionPolicy) mosaicRetentionPolicyJSON {
	return mosaicRetentionPolicyJSON{
		Version:     r.Version,
		CaptureMode: string(r.CaptureMode),
		Enforcement: r.Enforcement == config.PolicyEnforcementReject,
		Notes:       r.Notes,
	}
}

// RegisterPersistencePolicyTool registers mosaic_hexxla_get_persistence_policy (read-only; reflects startup-loaded YAML or defaults).
func RegisterPersistencePolicyTool(server *mcp.Server, rt config.MosaicRuntimeConfig, configFilePath string, log *slog.Logger, ratchetWrapper *RatchetWrapper) {
	type empty struct{}

	handler := func(ctx context.Context, req *mcp.CallToolRequest, _ empty) (*mcp.CallToolResult, MosaicConfigPolicyResponse, error) {
		return handlePersistencePolicy(ctx, req, rt, configFilePath, log)
	}

	if ratchetWrapper != nil {
		originalHandler := handler
		wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, _ empty) (*mcp.CallToolResult, MosaicConfigPolicyResponse, error) {
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
			if tokens, ok := session.Tokens[ratchetdomain.ToolName("mosaic_hexxla_get_persistence_policy")]; ok && len(tokens) > 0 {
				token = tokens[len(tokens)-1]
			}

			err = ratchetWrapper.ratchetSvc.ValidateToolCall(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_get_persistence_policy"), token)
			if err != nil {
				return nil, MosaicConfigPolicyResponse{}, fmt.Errorf("ratchet validation failed: %w", err)
			}

			result, resp, err := originalHandler(ctx, req, empty{})
			if err != nil {
				return result, resp, err
			}

			_, err = ratchetWrapper.ratchetSvc.IssueToken(ctx, sessionID, ratchetdomain.ToolName("mosaic_hexxla_get_persistence_policy"))
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
				session.RecordToolCall(ratchetdomain.ToolName("mosaic_hexxla_get_persistence_policy"))
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
		Name:        "mosaic_hexxla_get_persistence_policy",
		Description: "Return the Mosaic config snapshot loaded at server startup (YAML via -policy or MOSAIC_POLICY_FILE). Includes retention (capture_mode, enforcement as boolean, notes), allow_delete_cell, and optional config path. persistence_policy in YAML is deprecated in favor of retention.",
	}, handler)
}

func handlePersistencePolicy(ctx context.Context, _ *mcp.CallToolRequest, rt config.MosaicRuntimeConfig, configFilePath string, log *slog.Logger) (*mcp.CallToolResult, MosaicConfigPolicyResponse, error) {
	if log != nil {
		log.DebugContext(ctx, "mosaic_hexxla_get_persistence_policy invoked")
	}
	isDefault := configFilePath == ""
	out := MosaicConfigPolicyResponse{
		Version:         config.MosaicConfigVersion,
		Retention:       newMosaicRetentionPolicyJSON(rt.Retention),
		AllowDeleteCell: rt.AllowDeleteCell,
		ConfigFile:      configFilePath,
		IsDefaultConfig: isDefault,
	}
	return nil, out, nil
}
