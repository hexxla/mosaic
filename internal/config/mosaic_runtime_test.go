package config_test

import (
	"testing"

	"github.com/sploitzberg/mosaic/internal/config"
	"github.com/sploitzberg/mosaic/internal/core/domain"
)

func TestMosaicRuntimeConfig_AllowsPutCell(t *testing.T) {
	t.Parallel()
	pol := config.RetentionPolicy{
		Version:     1,
		CaptureMode: config.CaptureModeUserOnly,
		Enforcement: config.PolicyEnforcementReject,
	}
	g := config.NewMosaicRuntimeConfig(pol, false)
	if g.AllowsPutCell(domain.CellPutKindAssistantResponse) {
		t.Fatal("expected false for assistant under user_only+reject")
	}
	if !g.AllowsPutCell(domain.CellPutKindUserMessage) {
		t.Fatal("expected true for user message")
	}
}
