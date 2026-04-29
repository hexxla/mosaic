package services

import (
	"context"
	"fmt"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/ports/primary"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/ports/secondary"
)

// HealthService implements primary.Health by delegating to secondary.EngineHealth.
type HealthService struct {
	engine  secondary.EngineHealth
	version string
}

// NewHealthService constructs a HealthService. version is the mosaic binary version string
// included in [domain.HealthSummary.MosaicVersion].
func NewHealthService(engine secondary.EngineHealth, version string) *HealthService {
	return &HealthService{engine: engine, version: version}
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
	return summary, nil
}

// Ensure HealthService implements primary.Health.
var _ primary.Health = (*HealthService)(nil)
