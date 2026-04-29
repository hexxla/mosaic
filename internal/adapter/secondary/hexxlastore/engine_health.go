// Package hexxlastore implements secondary ports using github.com/hexxla/hexxladb (composition root opens DB).
package hexxlastore

import (
	"context"
	"fmt"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/ports/secondary"
)

// EngineHealthAdapter implements [secondary.EngineHealth] via (*hexxladb.DB).HealthCheck.
type EngineHealthAdapter struct {
	db *hexxladb.DB
}

// NewEngineHealthAdapter wraps an open database handle (caller owns lifecycle: Open/Close).
func NewEngineHealthAdapter(db *hexxladb.DB) *EngineHealthAdapter {
	return &EngineHealthAdapter{db: db}
}

// Check implements [secondary.EngineHealth].
func (a *EngineHealthAdapter) Check(ctx context.Context) (domain.HealthSummary, error) {
	if a == nil || a.db == nil {
		return domain.HealthSummary{}, fmt.Errorf("hexxlastore: nil database")
	}
	rep, err := a.db.HealthCheck(ctx, hexxladb.DefaultHealthCheckConfig())
	if err != nil {
		return domain.HealthSummary{}, fmt.Errorf("hexxladb health check: %w", err)
	}
	summary := mapHealthReport(&rep)
	summary.DatabaseLayout = domain.DatabaseLayout{
		PageSize:           a.db.PageSize(),
		MaxValueBytes:      a.db.MaxValueBytes(),
		EmbeddingDimension: a.db.EmbeddingDimension(),
		EmbeddingMetric:    embeddingMetricLabel(a.db.EmbeddingDimension(), a.db.EmbeddingMetric()),
	}
	summary.IntegrityOK = rep.TagIndexErrors == 0 && rep.SourceIndexErrors == 0 && len(rep.OrphanedSeams) == 0
	return summary, nil
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
