package hexxlastore

import (
	"context"
	"fmt"
	"os"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
)

// EngineHealthAdapter implements [secondary.EngineHealth] via (*hexxladb.DB).HealthCheck.
type EngineHealthAdapter struct {
	live        *LiveDB
	primaryPath string
}

// NewEngineHealthAdapter wraps a [LiveDB] (caller owns [LiveDB.Close]).
// primaryPath is the Hexxla primary file path used for on-disk size stats (see [domain.DiskFootprint]).
func NewEngineHealthAdapter(live *LiveDB, primaryPath string) *EngineHealthAdapter {
	return &EngineHealthAdapter{live: live, primaryPath: primaryPath}
}

// Check implements [secondary.EngineHealth].
func (a *EngineHealthAdapter) Check(ctx context.Context) (domain.HealthSummary, error) {
	if a == nil || a.live == nil {
		return domain.HealthSummary{}, fmt.Errorf("hexxlastore: nil database")
	}
	var summary domain.HealthSummary
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		rep, err := db.HealthCheck(ctx, hexxladb.DefaultHealthCheckConfig())
		if err != nil {
			return fmt.Errorf("hexxladb health check: %w", err)
		}
		summary = mapHealthReport(&rep)
		summary.DatabaseLayout = domain.DatabaseLayout{
			PageSize:           db.PageSize(),
			MaxValueBytes:      db.MaxValueBytes(),
			EmbeddingDimension: db.EmbeddingDimension(),
			EmbeddingMetric:    embeddingMetricLabel(db.EmbeddingDimension(), db.EmbeddingMetric()),
		}
		summary.Disk = diskFootprintFromPath(a.primaryPath)
		summary.IntegrityOK = rep.TagIndexErrors == 0 && rep.SourceIndexErrors == 0 && len(rep.OrphanedSeams) == 0
		return nil
	})
	return summary, err
}

func diskFootprintFromPath(primaryPath string) domain.DiskFootprint {
	out := domain.DiskFootprint{PrimaryPath: primaryPath}
	if primaryPath == "" {
		return out
	}
	if st, err := os.Stat(primaryPath); err == nil {
		out.PrimaryBytes = st.Size()
	}
	walPath := primaryPath + "-wal"
	if st, err := os.Stat(walPath); err == nil {
		out.WALBytes = st.Size()
	}
	out.TotalBytes = out.PrimaryBytes + out.WALBytes
	return out
}

func mapHealthReport(r *hexxladb.HealthReport) domain.HealthSummary {
	return domain.HealthSummary{
		CellCount:         r.CellCount,
		SeamCount:         r.SeamCount,
		SeamsResolved:     r.SeamsResolved,
		SeamsUnresolved:   r.SeamsUnresolved,
		OrphanSeamIDs:     append([]string(nil), r.OrphanedSeams...),
		TagIndexErrors:    r.TagIndexErrors,
		SourceIndexErrors: r.SourceIndexErrors,
		MVCC: domain.MVCCSnapshot{
			CommitSeq:     r.MVCCStats.CommitSeq,
			VersionedRows: r.MVCCStats.VersionedRows,
			LogicalCells:  r.MVCCStats.LogicalCells,
		},
		Warnings: append([]string(nil), r.Warnings...),
	}
}

func embeddingMetricLabel(dim uint16, m hexxladb.DistanceMetric) string {
	if dim == 0 {
		return ""
	}
	switch m {
	case hexxladb.DistanceCosine:
		return "cosine"
	case hexxladb.DistanceDotProduct:
		return "dot_product"
	case hexxladb.DistanceL2:
		return "l2"
	default:
		return "unknown"
	}
}

// Ensure EngineHealthAdapter implements secondary.EngineHealth.
var _ secondary.EngineHealth = (*EngineHealthAdapter)(nil)
