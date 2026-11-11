package services

import (
	"context"
	"fmt"

	"github.com/hexxla/mosaic/internal/core/domain"
	"github.com/hexxla/mosaic/internal/core/ports/primary"
	"github.com/hexxla/mosaic/internal/core/ports/secondary"
)

const (
	defaultContextMaxBudgetBytes = 4096
	defaultContextMaxRing        = 3
	minContextBudgetBytes        = domain.ContextBudgetMinBytes
	maxContextBudgetBytes        = domain.ContextBudgetMaxBytes
	maxContextMaxRing            = 32
	maxContextSeedCoords         = 32
	maxContextCandidateCellsCap  = 8192
)

// ContextAssemblyService implements [primary.ContextAssembly].
type ContextAssemblyService struct {
	loader secondary.ContextPackLoader
}

// NewContextAssemblyService constructs ContextAssemblyService.
func NewContextAssemblyService(loader secondary.ContextPackLoader) *ContextAssemblyService {
	return &ContextAssemblyService{loader: loader}
}

// LoadFromSeeds implements [primary.ContextAssembly].
func (s *ContextAssemblyService) LoadFromSeeds(ctx context.Context, cmd *domain.LoadContextPackCommand) (domain.ContextPackResponse, error) {
	if s == nil || s.loader == nil {
		return domain.ContextPackResponse{}, fmt.Errorf("context assembly: nil dependencies")
	}
	if cmd == nil {
		return domain.ContextPackResponse{}, fmt.Errorf("context assembly: nil command")
	}
	if len(cmd.Seeds) == 0 {
		return domain.ContextPackResponse{}, fmt.Errorf("context assembly: need at least one seed coordinate")
	}
	if len(cmd.Seeds) > maxContextSeedCoords {
		return domain.ContextPackResponse{}, fmt.Errorf("context assembly: too many seeds (max %d)", maxContextSeedCoords)
	}
	if cmd.Snapshot != nil {
		if err := validateSnapshot(*cmd.Snapshot); err != nil {
			return domain.ContextPackResponse{}, err
		}
		if cmd.EdgeFilter != "" || cmd.IncludeEdges {
			return domain.ContextPackResponse{}, fmt.Errorf("historical context cannot traverse or enrich current edges")
		}
	}
	normalized := *cmd
	if cmd.MaxRing < 0 || cmd.MaxCells < 0 || cmd.MaxHops < 0 || cmd.MaxHops > 32 || cmd.SeamRadius < 0 || cmd.SeamRadius > 32 {
		return domain.ContextPackResponse{}, fmt.Errorf("context assembly: ring, seam radius and hops must be 0..32; max_cells must be nonnegative")
	}

	if normalized.MaxRing <= 0 {
		normalized.MaxRing = defaultContextMaxRing
	}
	if normalized.MaxRing > maxContextMaxRing {
		normalized.MaxRing = maxContextMaxRing
	}
	effectiveBudget := normalized.MaxBudgetBytes
	if effectiveBudget == 0 {
		effectiveBudget = defaultContextMaxBudgetBytes
	}
	if effectiveBudget < minContextBudgetBytes || effectiveBudget > maxContextBudgetBytes {
		return domain.ContextPackResponse{}, fmt.Errorf("context assembly: resolved UTF-8 byte budget must be between %d and %d",
			minContextBudgetBytes, maxContextBudgetBytes)
	}
	if normalized.SessionByteLimit > 0 {
		effectiveBudget = min(effectiveBudget, normalized.SessionByteLimit)
		if effectiveBudget < minContextBudgetBytes {
			return domain.ContextPackResponse{}, fmt.Errorf("context assembly: remaining session budget is below %d bytes", minContextBudgetBytes)
		}
	}
	normalized.MaxBudgetBytes = effectiveBudget
	if normalized.MaxCells > maxContextCandidateCellsCap {
		normalized.MaxCells = maxContextCandidateCellsCap
	}
	out, err := s.loader.LoadFromSeeds(ctx, &normalized)
	if err != nil {
		return domain.ContextPackResponse{}, fmt.Errorf("context assembly: %w", err)
	}
	applyContextByteBudget(&out, effectiveBudget, normalized.Explain)
	out.SeedCount = len(normalized.Seeds)
	out.MaxRingApplied = normalized.MaxRing
	out.MaxBudgetBytes = effectiveBudget
	out.RetrievalHint = domain.RetrievalHintAfterContextPack
	return out, nil
}

// Ensure ContextAssemblyService implements primary.ContextAssembly.
var _ primary.ContextAssembly = (*ContextAssemblyService)(nil)
