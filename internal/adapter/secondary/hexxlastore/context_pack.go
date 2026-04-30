package hexxlastore

import (
	"context"
	"fmt"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
)

// ContextPackAdapter implements [secondary.ContextPackLoader] via Tx.LoadContextPackFrom.
type ContextPackAdapter struct {
	live *LiveDB
}

// NewContextPackAdapter wraps a [LiveDB] (caller owns [LiveDB.Close]).
func NewContextPackAdapter(live *LiveDB) *ContextPackAdapter {
	return &ContextPackAdapter{live: live}
}

// LoadFromSeeds implements [secondary.ContextPackLoader].
func (a *ContextPackAdapter) LoadFromSeeds(ctx context.Context, cmd *domain.LoadContextPackCommand) (domain.ContextPackResponse, error) {
	if a == nil || a.live == nil {
		return domain.ContextPackResponse{}, fmt.Errorf("hexxlastore context pack: nil database")
	}
	if cmd == nil || len(cmd.Seeds) == 0 {
		return domain.ContextPackResponse{}, fmt.Errorf("hexxlastore context pack: nil command or empty seeds")
	}
	coords := make([]hexxladb.Coord, 0, len(cmd.Seeds))
	for _, s := range cmd.Seeds {
		coords = append(coords, hexxladb.Coord{Q: s.Q, R: s.R})
	}
	cfg := hexxladb.LoadContextBudgetConfig{
		FilterSuperseded: cmd.FilterSuperseded,
		IncludeSeams:     cmd.IncludeSeams,
		Explain:          cmd.Explain,
		IncludeFacetText: cmd.IncludeFacetText,
		SeamRadius:       0,
	}
	cfg.Assemble = hexxladb.DefaultAssembleCellViewOpts()
	if cmd.MaxCells > 0 {
		cfg.MaxCandidateCells = cmd.MaxCells
	}

	var pack hexxladb.ContextPack
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		return db.View(func(tx *hexxladb.Tx) error {
			var errInner error
			pack, errInner = tx.LoadContextPackFrom(ctx, cmd.MaxRing, cmd.MaxTokens,
				hexxladb.ByteLenBudgeter{},
				cfg,
				coords...,
			)
			return errInner
		})
	})
	if err != nil {
		return domain.ContextPackResponse{}, fmt.Errorf("hexxlastore load context pack: %w", err)
	}

	out := domain.ContextPackResponse{
		Cells:           make([]domain.ContextPackCell, 0, len(pack.Cells)),
		TotalTokens:     pack.TotalTokens,
		SeedCount:       len(cmd.Seeds),
		MaxRingApplied:  cmd.MaxRing,
		MaxTokensBudget: cmd.MaxTokens,
		Stats: domain.ContextPackStatsDTO{
			CandidatesScanned: pack.Stats.CandidatesScanned,
			CellsEvicted:      pack.Stats.CellsEvicted,
			MaxRingUsed:       pack.Stats.MaxRingUsed,
		},
	}
	for i := range pack.Cells {
		v := &pack.Cells[i]
		createdAt, updatedAt, validFrom, validTo := timingsFromCellView(v)
		out.Cells = append(out.Cells, domain.ContextPackCell{
			Coord:      domain.AxialCoord{Q: v.Coord.Q, R: v.Coord.R},
			RawContent: v.RawContent,
			Tags:       append([]string(nil), v.Tags...),
			SourceID:   v.Provenance.SourceID,
			Confidence: v.Provenance.Confidence,
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
			ValidFrom:  validFrom,
			ValidTo:    validTo,
		})
	}
	out.Explanations = formatContextExplanations(pack.Explanations)
	out.Seams = seamSummariesFromPack(&pack)

	return out, nil
}

func formatContextExplanations(exps []hexxladb.CellExplanation) []string {
	if len(exps) == 0 {
		return nil
	}
	out := make([]string, 0, len(exps))
	for _, e := range exps {
		out = append(out, fmt.Sprintf("(%d,%d) ring=%d %s tokens=%d",
			e.Coord.Q, e.Coord.R, e.Ring, e.Reason, e.Tokens))
	}
	return out
}

// seamSummariesFromPack maps seam records from an assembled pack.
func seamSummariesFromPack(pack *hexxladb.ContextPack) []domain.ContextSeamSummary {
	if pack == nil {
		return nil
	}
	n := len(pack.Seams)
	if n == 0 {
		return nil
	}
	out := make([]domain.ContextSeamSummary, 0, n)
	for i := range pack.Seams {
		s := pack.Seams[i]
		out = append(out, domain.ContextSeamSummary{
			ID:               s.ID,
			SeamType:         s.SeamType,
			Reason:           s.Reason,
			ResolutionStatus: s.ResolutionStatus,
		})
	}
	return out
}

// Ensure ContextPackAdapter implements secondary.ContextPackLoader.
var _ secondary.ContextPackLoader = (*ContextPackAdapter)(nil)
