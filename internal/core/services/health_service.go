package services

import (
	"context"
	"fmt"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
)

// HealthService implements primary.Health by delegating to secondary.EngineHealth.
type HealthService struct {
	engine                      secondary.EngineHealth
	version                     string
	mvccRetainCommitsBehindHead uint64
}

// NewHealthService constructs a HealthService. version is the mosaic binary version string
// included in [domain.HealthSummary.MosaicVersion].
// mvccRetainCommitsBehindHead is the effective Mosaic YAML/database retention forwarded at open (including injected defaults).
func NewHealthService(engine secondary.EngineHealth, version string, mvccRetainCommitsBehindHead uint64) *HealthService {
	return &HealthService{
		engine:                      engine,
		version:                     version,
		mvccRetainCommitsBehindHead: mvccRetainCommitsBehindHead,
	}
}

// Status implements [primary.Health].
func (s *HealthService) Status(ctx context.Context) (domain.HealthSummary, error) {
	if s.engine == nil {
		return domain.HealthSummary{}, fmt.Errorf("health: nil engine")
	}
	summary, err := s.engine.Check(ctx)
	if err != nil {
		return domain.HealthSummary{}, fmt.Errorf("health: %w", err)
	}
	summary.MosaicVersion = s.version
	summary.MVCCRetainCommitsBehindHead = s.mvccRetainCommitsBehindHead
	return summary, nil
}

// Ensure HealthService implements primary.Health.
var _ primary.Health = (*HealthService)(nil)
