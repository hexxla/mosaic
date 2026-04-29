package config_test

import (
	"strings"
	"testing"

	"github.com/sploitzberg/go-llm-project-structure/internal/config"
)

func TestMCPPolicyInstructions_defaultNoFile(t *testing.T) {
	t.Parallel()
	rt := config.NewMosaicRuntimeConfig(config.DefaultRetentionPolicy(), false)
	s := config.MCPPolicyInstructions(rt, "")
	if !strings.Contains(s, "in-process defaults") {
		t.Fatalf("expected in-process defaults mention: %q", s)
	}
	if !strings.Contains(s, "not allowed") {
		t.Fatalf("expected delete not allowed: %q", s)
	}
	if !strings.Contains(s, "Retention enforcement: false") {
		t.Fatalf("expected enforcement false line: %q", s)
	}
}

func TestMCPPolicyInstructions_configFilePath(t *testing.T) {
	t.Parallel()
	rt := config.NewMosaicRuntimeConfig(config.DefaultRetentionPolicy(), true)
	s := config.MCPPolicyInstructions(rt, "/tmp/mosaic.yaml")
	if !strings.Contains(s, `"/tmp/mosaic.yaml"`) {
		t.Fatalf("expected file path in output: %q", s)
	}
	if !strings.Contains(s, "allowed (allow_delete_cell: true)") {
		t.Fatalf("expected delete allowed: %q", s)
	}
}

func TestMCPPolicyInstructions_enforcementRejectAndNotes(t *testing.T) {
	t.Parallel()
	rt := config.NewMosaicRuntimeConfig(config.RetentionPolicy{
		Version:     config.MosaicConfigVersion,
		CaptureMode: config.CaptureModeUserOnly,
		Enforcement: config.PolicyEnforcementReject,
		Notes:       "Keep assistant turns out of DB.",
	}, false)
	s := config.MCPPolicyInstructions(rt, "")
	if !strings.Contains(s, "Retention enforcement: true") {
		t.Fatalf("expected enforcement true line: %q", s)
	}
	if !strings.Contains(s, "returns an error for put_cell kinds") {
		t.Fatalf("expected updated enforcement wording: %q", s)
	}
	if !strings.Contains(s, "Keep assistant turns out of DB.") {
		t.Fatalf("expected notes: %q", s)
	}
	if !strings.Contains(s, "user_only") {
		t.Fatalf("expected capture_mode in output: %q", s)
	}
}
