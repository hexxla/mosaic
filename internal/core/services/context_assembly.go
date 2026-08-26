package services

import (
	"context"
	"fmt"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
)

const (
	defaultContextMaxTokens     = 4096
	defaultContextMaxRing       = 3
	minContextMaxTokens         = domain.ContextBudgetMinBytes
	maxContextMaxTokens         = domain.ContextBudgetMaxBytes
	maxContextMaxRing           = 32
	maxContextSeedCoords        = 32
	maxContextCandidateCellsCap = 8192
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
	normalized := *cmd
	if normalized.MaxRing <= 0 {
		normalized.MaxRing = defaultContextMaxRing
	}
	if normalized.MaxRing > maxContextMaxRing {
		normalized.MaxRing = maxContextMaxRing
	}
	effectiveBudget, err := resolveContextByteBudget(&normalized)
	if err != nil {
		return domain.ContextPackResponse{}, err
	}
	if effectiveBudget < minContextMaxTokens || effectiveBudget > maxContextMaxTokens {
		return domain.ContextPackResponse{}, fmt.Errorf("context assembly: resolved UTF-8 byte budget must be between %d and %d",
			minContextMaxTokens, maxContextMaxTokens)
	}
	normalized.MaxTokens = effectiveBudget
	normalized.OmitBudget = false
	normalized.BudgetTokensApprox = 0
	normalized.BytesPerApproxToken = 0
	normalized.MaxBudgetBytes = 0
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
	out.MaxTokensBudget = effectiveBudget
	out.RetrievalHint = domain.RetrievalHintAfterContextPack
	return out, nil
}

// Ensure ContextAssemblyService implements primary.ContextAssembly.
var _ primary.ContextAssembly = (*ContextAssemblyService)(nil)
