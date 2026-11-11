package config_test

import (
	"testing"

	"github.com/hexxla/mosaic/internal/config"
	"github.com/hexxla/mosaic/internal/core/domain"
)

func TestMosaicRuntimeConfigMutationPolicy(t *testing.T) {
	t.Parallel()
	pol := config.RetentionPolicy{
		Version:     1,
		CaptureMode: config.CaptureModeUserOnly,
		Enforcement: config.PolicyEnforcementReject,
	}
	g := config.NewMosaicRuntimeConfig(pol, false)
	if g.MutationPolicy().PutCellDenied(domain.CellPutKindAssistantResponse) == nil {
		t.Fatal("expected false for assistant under user_only+reject")
	}
	if g.MutationPolicy().PutCellDenied(domain.CellPutKindUserMessage) != nil {
		t.Fatal("expected true for user message")
	}
}
