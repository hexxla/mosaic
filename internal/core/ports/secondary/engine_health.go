package secondary

import (
	"context"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// EngineHealth is the driven port for running a storage engine health check.
type EngineHealth interface {
	Check(ctx context.Context) (domain.HealthSummary, error)
}
