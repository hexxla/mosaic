package config

import (
	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// MosaicRuntimeConfig aggregates startup-loaded settings used to gate mutating MCP tools
// and [services.CellMutationService]. Extend as config.yaml grows.
type MosaicRuntimeConfig struct {
	Retention       RetentionPolicy
	AllowDeleteCell bool
}

// NewMosaicRuntimeConfig wraps loaded YAML (or defaults).
func NewMosaicRuntimeConfig(r RetentionPolicy, allowDeleteCell bool) MosaicRuntimeConfig {
	return MosaicRuntimeConfig{
		Retention:       r,
		AllowDeleteCell: allowDeleteCell,
	}
}

// PutCellDenied returns nil if allowed, otherwise the retention enforcement error.
func (c MosaicRuntimeConfig) PutCellDenied(kind domain.CellPutKind) error {
	return (&c.Retention).CheckPutCell(kind)
}

// MutationPolicy maps loaded configuration into the application write policy.
func (c MosaicRuntimeConfig) MutationPolicy() domain.MutationPolicy {
	return domain.MutationPolicy{
		UserMessageDenied:       c.PutCellDenied(domain.CellPutKindUserMessage),
		AssistantResponseDenied: c.PutCellDenied(domain.CellPutKindAssistantResponse),
		AllowDeleteCell:         c.AllowDeleteCell,
	}
}
