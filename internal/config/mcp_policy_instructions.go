package config

import (
	"fmt"
	"strings"
)

// MCPPolicyInstructions returns text for the MCP server's initialize-instructions field so connected
// clients can inject the active persistence policy into the model context without manual rule edits
// when the YAML changes.
func MCPPolicyInstructions(rt MosaicRuntimeConfig, configFilePath string) string {
	var b strings.Builder
	b.WriteString("Mosaic persistence policy (active for this MCP server; loaded at startup).\n")
	b.WriteString("Follow this when calling mosaic_hexxla_put_cell, mosaic_hexxla_delete_cell, and related mutating tools.\n\n")

	if strings.TrimSpace(configFilePath) == "" {
		b.WriteString("Config source: in-process defaults (no YAML file; set -policy or MOSAIC_POLICY_FILE to load a file).\n")
	} else {
		fmt.Fprintf(&b, "Config source: file %q\n", configFilePath)
	}

	r := rt.Retention
	fmt.Fprintf(&b, "Retention capture_mode: %s\n", r.CaptureMode)
	if r.Enforcement == PolicyEnforcementReject {
		fmt.Fprintf(&b, "Retention enforcement: true — server rejects put_cell kinds that conflict with capture_mode.\n")
	} else {
		fmt.Fprintf(&b, "Retention enforcement: false — capture_mode is advisory only; server does not block puts by kind (allow_delete_cell is separate).\n")
	}
	if strings.TrimSpace(r.Notes) != "" {
		fmt.Fprintf(&b, "Retention notes: %s\n", strings.TrimSpace(r.Notes))
	}

	if rt.AllowDeleteCell {
		b.WriteString("Delete cells: mosaic_hexxla_delete_cell is allowed (allow_delete_cell: true).\n")
	} else {
		b.WriteString("Delete cells: mosaic_hexxla_delete_cell is not allowed (allow_delete_cell defaults to false without an explicit true in config).\n")
	}

	return b.String()
}
