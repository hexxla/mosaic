package services

import (
	"context"
	"fmt"
	"time"

	"github.com/hexxla/mosaic/internal/core/domain"
	"github.com/hexxla/mosaic/internal/core/ports/secondary"
)

// MemoryExplorationService applies per-operation limits before native exploration.
type MemoryExplorationService struct{ reader secondary.MemoryExplorer }

// NewMemoryExplorationService constructs advanced read workflows.
func NewMemoryExplorationService(reader secondary.MemoryExplorer) *MemoryExplorationService {
	return &MemoryExplorationService{reader: reader}
}

func normalizeGraph(q domain.GraphQuery) (domain.GraphQuery, error) {
	if q.MaxHops < 0 || q.MaxHops > 16 || q.MaxNodes < 0 || q.MaxNodes > 256 || q.MaxEdgeRows < 0 || q.MaxEdgeRows > 10000 || len(q.Relation) > 1024 {
		return q, fmt.Errorf("graph limits: hops 0..16, nodes 0..256, edge rows 0..10000, relation at most 1024 bytes")
	}
	if q.MaxHops == 0 {
		q.MaxHops = 5
	}
	if q.MaxNodes == 0 {
		q.MaxNodes = 100
	}
	if q.MaxEdgeRows == 0 {
		q.MaxEdgeRows = 5000
	}
	return q, nil
}

// WalkGraph explores a bounded outgoing neighborhood.
func (s *MemoryExplorationService) WalkGraph(ctx context.Context, q domain.GraphQuery) (domain.CoordinateResults, error) {
	if q.Goal != nil {
		return domain.CoordinateResults{}, fmt.Errorf("walk_graph does not accept goal")
	}
	q, err := normalizeGraph(q)
	if err != nil {
		return domain.CoordinateResults{}, err
	}
	return s.reader.WalkGraph(ctx, q)
}

// FindPath follows native weighted directed-edge semantics.
func (s *MemoryExplorationService) FindPath(ctx context.Context, q domain.GraphQuery) (domain.CoordinateResults, error) {
	if q.MaxHops != 0 {
		return domain.CoordinateResults{}, fmt.Errorf("max_hops applies only to walk_graph; path search uses max_nodes and max_edge_rows")
	}
	q, err := normalizeGraph(q)
	if err != nil {
		return domain.CoordinateResults{}, err
	}
	if q.Goal == nil {
		return domain.CoordinateResults{}, fmt.Errorf("path requires goal")
	}
	return s.reader.FindPath(ctx, q)
}

func normalizeSpatial(q domain.SpatialQuery) (domain.SpatialQuery, error) {
	if len(q.Seeds) == 0 || len(q.Seeds) > 8 || q.Radius < 0 || q.Radius > 16 || q.MaxCells < 0 || q.MaxCells > 100 || len(q.Blocked) > 1024 {
		return q, fmt.Errorf("spatial limits: 1..8 seeds, radius 0..16, cells 0..100, at most 1024 blockers")
	}
	if q.Radius == 0 {
		q.Radius = 4
	}
	if q.MaxCells == 0 {
		q.MaxCells = 100
	}
	return q, nil
}

// LoadFOV loads one seed through stable explicit occluders.
func (s *MemoryExplorationService) LoadFOV(ctx context.Context, q domain.SpatialQuery) (domain.SpatialResults, error) {
	q, err := normalizeSpatial(q)
	if err != nil {
		return domain.SpatialResults{}, err
	}
	if len(q.Seeds) != 1 {
		return domain.SpatialResults{}, fmt.Errorf("FOV requires one seed")
	}
	return s.reader.LoadFOV(ctx, q)
}

// LoadVoronoi assigns cells to bounded seed regions.
func (s *MemoryExplorationService) LoadVoronoi(ctx context.Context, q domain.SpatialQuery) (domain.SpatialResults, error) {
	q, err := normalizeSpatial(q)
	if err != nil {
		return domain.SpatialResults{}, err
	}
	if len(q.Blocked) > 0 {
		return domain.SpatialResults{}, fmt.Errorf("voronoi does not accept FOV blockers")
	}
	return s.reader.LoadVoronoi(ctx, q)
}

// Density reports occupancy around one seed.
func (s *MemoryExplorationService) Density(ctx context.Context, q domain.SpatialQuery) (domain.DensityResults, error) {
	q, err := normalizeSpatial(q)
	if err != nil {
		return domain.DensityResults{}, err
	}
	if len(q.Seeds) != 1 {
		return domain.DensityResults{}, fmt.Errorf("density requires one seed")
	}
	return s.reader.Density(ctx, q)
}

// RenderGrid renders a small operator-readable neighborhood.
func (s *MemoryExplorationService) RenderGrid(ctx context.Context, q domain.SpatialQuery) (domain.GridResult, error) {
	q, err := normalizeSpatial(q)
	if err != nil {
		return domain.GridResult{}, err
	}
	if len(q.Seeds) != 1 || q.Radius > 10 {
		return domain.GridResult{}, fmt.Errorf("grid requires one seed and radius at most 10")
	}
	return s.reader.RenderGrid(ctx, q)
}

func normalizeTaxonomy(q domain.TaxonomyQuery) (domain.TaxonomyQuery, error) {
	if q.Radius < 0 || q.Radius > 16 || q.MinCount < 0 || q.MaxResults < 0 || q.MaxResults > 100 || q.MaxScanRows < 0 || q.MaxScanRows > 20000 {
		return q, fmt.Errorf("taxonomy limits: radius 0..16, results 0..100, scan rows 0..20000, nonnegative min_count")
	}
	if q.MinCount == 0 {
		q.MinCount = 1
	}
	if q.MaxResults == 0 {
		q.MaxResults = 100
	}
	if q.MaxScanRows == 0 {
		q.MaxScanRows = 2000
	}
	return q, nil
}

// TagPairs reports co-occurrence within an explicitly bounded cell sample.
func (s *MemoryExplorationService) TagPairs(ctx context.Context, q domain.TaxonomyQuery) (domain.TaxonomyResults, error) {
	q, err := normalizeTaxonomy(q)
	if err != nil {
		return domain.TaxonomyResults{}, err
	}
	return s.reader.TagPairs(ctx, q)
}

// Untagged finds regional cells needing taxonomy maintenance.
func (s *MemoryExplorationService) Untagged(ctx context.Context, q domain.TaxonomyQuery) (domain.TaxonomyResults, error) {
	q, err := normalizeTaxonomy(q)
	if err != nil {
		return domain.TaxonomyResults{}, err
	}
	return s.reader.Untagged(ctx, q)
}

// ListSnapshots inspects names without mutating retention.
func (s *MemoryExplorationService) ListSnapshots(ctx context.Context) (domain.SnapshotLabels, error) {
	return s.reader.ListSnapshots(ctx)
}

// GetHistoricalCell reads a cell from one retained snapshot.
func (s *MemoryExplorationService) GetHistoricalCell(ctx context.Context, q domain.HistoricalCellQuery) (domain.CellLookupResponse, error) {
	if err := validateSnapshot(q.Snapshot); err != nil {
		return domain.CellLookupResponse{}, err
	}
	return s.reader.GetHistoricalCell(ctx, q)
}
func validateSnapshot(q domain.SnapshotSelector) error {
	if err := validateWireTime(q.At); err != nil {
		return err
	}
	n := 0
	if q.Seq != nil {
		n++
	}
	if q.At != nil {
		n++
	}
	if q.Tag != "" {
		n++
	}
	if n != 1 || len(q.Tag) > 256 {
		return fmt.Errorf("snapshot requires exactly one of seq, tag, or at; tag at most 256 bytes")
	}
	return nil
}

// DiffHistory returns a narrow diagnostic diff; native collection is materialized.
func (s *MemoryExplorationService) DiffHistory(ctx context.Context, q domain.HistoryDiffQuery) (domain.HistoryDiffResults, error) {
	if q.FromSeq > q.ToSeq || q.ToSeq-q.FromSeq > 64 || q.MaxResults < 0 || q.MaxResults > 100 {
		return domain.HistoryDiffResults{}, fmt.Errorf("history diff requires ordered window of at most 64 commits and at most 100 results")
	}
	if q.MaxResults == 0 {
		q.MaxResults = 100
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.reader.DiffHistory(ctx, q)
}

// ReadChanges scans a bounded record window, then applies operation filters.
func (s *MemoryExplorationService) ReadChanges(ctx context.Context, q domain.ChangesQuery) (domain.ChangesResults, error) {
	if q.MaxRecords < 0 || q.MaxRecords > 128 || len(q.Operations) > 6 {
		return domain.ChangesResults{}, fmt.Errorf("changefeed limits: up to 128 records and 6 operation filters")
	}
	if q.MaxRecords == 0 {
		q.MaxRecords = 32
	}
	if q.IncludePayload && q.MaxRecords > 32 {
		return domain.ChangesResults{}, fmt.Errorf("inline payload inspection supports at most 32 records")
	}
	for _, op := range q.Operations {
		switch op {
		case "put_cell", "delete_cell", "put_facet", "put_edge", "put_seam", "resolve_seam":
		default:
			return domain.ChangesResults{}, fmt.Errorf("unknown changefeed operation %q", op)
		}
	}
	return s.reader.ReadChanges(ctx, q)
}

// GetVector inspects a current embedding without invoking a model.
func (s *MemoryExplorationService) GetVector(ctx context.Context, q domain.AxialCoord) (domain.VectorResult, error) {
	return s.reader.GetVector(ctx, q)
}

// Diagnostics reports native storage accounting and cumulative write timings.
func (s *MemoryExplorationService) Diagnostics(ctx context.Context) (domain.DiagnosticsResult, error) {
	return s.reader.Diagnostics(ctx)
}

// EmbeddingMaintenanceService bounds derived-index work and vector deletion.
type EmbeddingMaintenanceService struct {
	store   secondary.EmbeddingMaintenance
	enabled bool
}

// NewEmbeddingMaintenanceService constructs model-compatible vector maintenance.
func NewEmbeddingMaintenanceService(store secondary.EmbeddingMaintenance, enabled bool) *EmbeddingMaintenanceService {
	return &EmbeddingMaintenanceService{store: store, enabled: enabled}
}

// DeleteVector removes a vector while preserving the live cell.
func (s *EmbeddingMaintenanceService) DeleteVector(ctx context.Context, q domain.AxialCoord) (domain.MutationOK, error) {
	if !s.enabled {
		return domain.MutationOK{}, fmt.Errorf("embedding maintenance disabled in lexical-only mode")
	}
	return s.store.DeleteVector(ctx, q)
}

// RebuildIndex rebuilds derived HNSW state with conservative resource ceilings.
func (s *EmbeddingMaintenanceService) RebuildIndex(ctx context.Context, q domain.RebuildIndexQuery) (domain.RebuildIndexResult, error) {
	if !s.enabled {
		return domain.RebuildIndexResult{}, fmt.Errorf("embedding maintenance disabled in lexical-only mode")
	}
	if q.MaxVectors < 0 || q.MaxVectors > 10000 || q.MaxMemoryBytes > 512<<20 {
		return domain.RebuildIndexResult{}, fmt.Errorf("index rebuild limits: at most 10000 vectors and 512 MiB")
	}
	if q.MaxVectors == 0 {
		q.MaxVectors = 1000
	}
	if q.MaxMemoryBytes == 0 {
		q.MaxMemoryBytes = 128 << 20
	}
	return s.store.RebuildIndex(ctx, q)
}
