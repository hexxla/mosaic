package hexxlastore

import (
	"context"
	"fmt"
	"slices"
	"time"

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
	assemble.IncludeEdges = cmd.IncludeEdges
	assemble.UnresolvedSeamsOnly = cmd.UnresolvedSeamsOnly
	cfg := hexxladb.LoadContextConfig{
		Seeds:      coords,
		EdgeFilter: cmd.EdgeFilter, MaxHops: cmd.MaxHops, AsOf: cmd.AsOf,
		MaxRing:  cmd.MaxRing,
		MaxCells: cmd.MaxCells,
		Assembly: hexxladb.ContextAssemblyConfig{
			Assemble:         assemble,
			SeamRadius:       cmd.SeamRadius,
			FilterSuperseded: cmd.FilterSuperseded,
			IncludeSeams:     cmd.IncludeSeams,
			Explain:          cmd.Explain,
		},
	}

	var pack hexxladb.ContextPack
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		return viewSnapshot(db, cmd.Snapshot, func(tx *hexxladb.Tx) error {
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
		var facets []domain.ContextFacetMetadata
		var edges []domain.EdgeBullet
		budgetBytes := len(v.RawContent)
		for _, facet := range v.Facets {
			facetText = append(facetText, facet.DerivedContent)
			budgetBytes += len(facet.DerivedContent)
			rotated := ""
			if !facet.LastRotated.IsZero() {
				rotated = facet.LastRotated.UTC().Format(time.RFC3339Nano)
			}
			facets = append(facets, domain.ContextFacetMetadata{FacetID: facet.ID, LastRotated: rotated, DerivationHash: facet.DerivationHash})
		}
		for _, edge := range v.Edges {
			edges = append(edges, domain.EdgeBullet{ToQ: edge.To.Q, ToR: edge.To.R, RelationType: edge.RelationType, Weight: edge.Weight})
			budgetBytes += len(edge.RelationType)
		}
		budgetCoord := v.Coord
		if v.SupersededFrom != nil {
			budgetCoord = *v.SupersededFrom
		}
		out.Cells = append(out.Cells, domain.ContextPackCell{
			ClusterHint: domainCoordPtr(v.ClusterHint), SupersededFrom: domainCoordPtr(v.SupersededFrom), FacetMetadata: facets, Edges: edges,
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
	out.Seams, err = seamSummariesFromPack(&pack)
	if err != nil {
		return domain.ContextPackResponse{}, err
	}
	if cmd.UnresolvedSeamsOnly {
		out.Seams = slices.DeleteFunc(out.Seams, func(s domain.ContextSeamSummary) bool { return s.ResolutionStatus != "" })
	}

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
func seamSummariesFromPack(pack *hexxladb.ContextPack) ([]domain.ContextSeamSummary, error) {
	if pack == nil {
		return nil, nil
	}
	n := len(pack.Seams)
	if n == 0 {
		return nil, nil
	}
	out := make([]domain.ContextSeamSummary, 0, n)
	for i := range pack.Seams {
		s := pack.Seams[i]
		a, err := hexxladb.Unpack(s.CellA)
		if err != nil {
			return nil, err
		}
		b, err := hexxladb.Unpack(s.CellB)
		if err != nil {
			return nil, err
		}
		out = append(out, domain.ContextSeamSummary{
			CellA: domain.AxialCoord(a), CellB: domain.AxialCoord(b), ResolutionNote: s.ResolutionNote, SourceID: s.Provenance.SourceID,
			ValidFrom: nanoWallPtrToRFC3339(s.Validity.ValidFrom), ValidTo: nanoWallPtrToRFC3339(s.Validity.ValidTo),
			ID:               s.ID,
			SeamType:         s.SeamType,
			Reason:           s.Reason,
			ResolutionStatus: s.ResolutionStatus,
		})
	}
	return out, nil
}

// Ensure ContextPackAdapter implements secondary.ContextPackLoader.
var _ secondary.ContextPackLoader = (*ContextPackAdapter)(nil)
