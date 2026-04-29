package mcpsrv

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/go-llm-project-structure/internal/config"
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
func RegisterPersistencePolicyTool(server *mcp.Server, rt config.MosaicRuntimeConfig, configFilePath string, log *slog.Logger) {
	type empty struct{}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_get_persistence_policy",
		Description: "Return the Mosaic config snapshot loaded at server startup (YAML via -policy or MOSAIC_POLICY_FILE). Includes retention (capture_mode, enforcement as boolean, notes), allow_delete_cell, and optional config path. persistence_policy in YAML is deprecated in favor of retention.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ empty) (*mcp.CallToolResult, MosaicConfigPolicyResponse, error) {
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
	})
}
