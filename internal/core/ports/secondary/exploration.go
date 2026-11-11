package secondary

import (
	"context"

	"github.com/hexxla/mosaic/internal/core/domain"
)

// MemoryExplorer reads bounded graph, spatial and retained-history views.
type MemoryExplorer interface {
	WalkGraph(context.Context, domain.GraphQuery) (domain.CoordinateResults, error)
	FindPath(context.Context, domain.GraphQuery) (domain.CoordinateResults, error)
	LoadFOV(context.Context, domain.SpatialQuery) (domain.SpatialResults, error)
	LoadVoronoi(context.Context, domain.SpatialQuery) (domain.SpatialResults, error)
	Density(context.Context, domain.SpatialQuery) (domain.DensityResults, error)
	RenderGrid(context.Context, domain.SpatialQuery) (domain.GridResult, error)
	TagPairs(context.Context, domain.TaxonomyQuery) (domain.TaxonomyResults, error)
	Untagged(context.Context, domain.TaxonomyQuery) (domain.TaxonomyResults, error)
	ListSnapshots(context.Context) (domain.SnapshotLabels, error)
	GetHistoricalCell(context.Context, domain.HistoricalCellQuery) (domain.CellLookupResponse, error)
	DiffHistory(context.Context, domain.HistoryDiffQuery) (domain.HistoryDiffResults, error)
	ReadChanges(context.Context, domain.ChangesQuery) (domain.ChangesResults, error)
	GetVector(context.Context, domain.AxialCoord) (domain.VectorResult, error)
	Diagnostics(context.Context) (domain.DiagnosticsResult, error)
}

// EmbeddingMaintenance changes vectors or their derived index, never model identity.
type EmbeddingMaintenance interface {
	DeleteVector(context.Context, domain.AxialCoord) (domain.MutationOK, error)
	RebuildIndex(context.Context, domain.RebuildIndexQuery) (domain.RebuildIndexResult, error)
}
