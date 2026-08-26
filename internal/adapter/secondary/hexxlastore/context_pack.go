package hexxlastore

import (
	"context"
	"fmt"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
)

// ContextPackAdapter implements [secondary.ContextPackLoader] via Tx.LoadContext.
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
	assemble := hexxladb.DefaultAssembleCellViewOpts()
	assemble.IncludeFacets = cmd.IncludeFacetText
	cfg := hexxladb.LoadContextConfig{
		Seeds:    coords,
		MaxRing:  cmd.MaxRing,
		MaxCells: cmd.MaxCells,
		Assembly: hexxladb.ContextAssemblyConfig{
			Assemble:         assemble,
			FilterSuperseded: cmd.FilterSuperseded,
			IncludeSeams:     cmd.IncludeSeams,
			Explain:          cmd.Explain,
		},
	}

	var pack hexxladb.ContextPack
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		return db.View(func(tx *hexxladb.Tx) error {
			var errInner error
			pack, errInner = tx.LoadContext(ctx, cfg)
			return errInner
		})
	})
	if err != nil {
		return domain.ContextPackResponse{}, fmt.Errorf("hexxlastore load context pack: %w", err)
	}

	out := domain.ContextPackResponse{
		Cells: make([]domain.ContextPackCell, 0, len(pack.Cells)),
		Stats: domain.ContextPackStatsDTO{
			CandidatesScanned: pack.Stats.CandidatesScanned,
			MaxRingUsed:       pack.Stats.MaxRingUsed,
		},
	}
	for i := range pack.Cells {
		v := &pack.Cells[i]
		createdAt, updatedAt, validFrom, validTo := timingsFromCellView(v)
		facetText := make([]string, 0, len(v.Facets))
		budgetBytes := len(v.RawContent)
		for _, facet := range v.Facets {
			facetText = append(facetText, facet.DerivedContent)
			budgetBytes += len(facet.DerivedContent)
		}
		budgetCoord := v.Coord
		if v.SupersededFrom != nil {
			budgetCoord = *v.SupersededFrom
		}
		out.Cells = append(out.Cells, domain.ContextPackCell{
			Coord:       domain.AxialCoord{Q: v.Coord.Q, R: v.Coord.R},
			RawContent:  v.RawContent,
			Tags:        append([]string(nil), v.Tags...),
			SourceID:    v.Provenance.SourceID,
			Confidence:  v.Provenance.Confidence,
			FacetText:   facetText,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
			ValidFrom:   validFrom,
			ValidTo:     validTo,
			BudgetBytes: budgetBytes,
			BudgetRing:  nearestSeedRing(budgetCoord, coords),
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
		if e.Reason != "superseded" {
			continue
		}
		if e.SupersededBy != nil {
			out = append(out, fmt.Sprintf("(%d,%d) ring=%d superseded by=(%d,%d)",
				e.Coord.Q, e.Coord.R, e.Ring, e.SupersededBy.Q, e.SupersededBy.R))
			continue
		}
		out = append(out, fmt.Sprintf("(%d,%d) ring=%d superseded", e.Coord.Q, e.Coord.R, e.Ring))
	}
	return out
}

func nearestSeedRing(coord hexxladb.Coord, seeds []hexxladb.Coord) int {
	best := 0
	for i, seed := range seeds {
		distance := coord.Distance(seed)
		if i == 0 || distance < best {
			best = distance
		}
	}
	return best
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
