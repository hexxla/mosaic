package config

import (
	"errors"

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

// AllowsPutCell reports whether put_cell for kind is permitted under retention policy.
func (c MosaicRuntimeConfig) AllowsPutCell(kind domain.CellPutKind) bool {
	return c.PutCellDenied(kind) == nil
}

// PutCellDenied returns nil if allowed, otherwise the retention enforcement error.
func (c MosaicRuntimeConfig) PutCellDenied(kind domain.CellPutKind) error {
	return (&c.Retention).CheckPutCell(kind)
}

// AllowsDeleteCell reports whether mosaic_hexxla_delete_cell is permitted (config allow_delete_cell).
func (c MosaicRuntimeConfig) AllowsDeleteCell() bool {
	return c.AllowDeleteCell
}

// DeleteCellDenied returns nil if delete is allowed, otherwise an error (default: deletes disabled).
func (c MosaicRuntimeConfig) DeleteCellDenied() error {
	if c.AllowDeleteCell {
		return nil
	}
	return errors.New("mosaic config: delete_cell not allowed (allow_delete_cell is false)")
}
