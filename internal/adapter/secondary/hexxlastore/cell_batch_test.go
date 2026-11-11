package hexxlastore

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/hexxla/hexxladb"

	"github.com/hexxla/mosaic/internal/config"
	"github.com/hexxla/mosaic/internal/core/domain"
)

func TestAtomicBatchReplayAndRollback(t *testing.T) {
	db, err := hexxladb.Open(filepath.Join(t.TempDir(), "batch.hexxla"), &hexxladb.Options{EmbeddingDimension: 2})
	if err != nil {
		t.Fatal(err)
	}
	live := NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })
	writer := NewCellWriterAdapter(live, config.DeleteAutoMaintainConfig{})
	cmd := &domain.PutCellsCommand{Cells: []domain.PutCellCommand{
		{RawContent: "first", SourceID: "test", Placement: domain.CellPlacementNearAnchor, MaxRadius: 2, IdempotencyKey: "first"},
		{RawContent: "second", SourceID: "test", Placement: domain.CellPlacementNearAnchor, MaxRadius: 2, IdempotencyKey: "second"},
	}}
	if _, err := writer.PutCells(t.Context(), cmd, [][]float32{nil, {1}}); !errors.Is(err, hexxladb.ErrEmbeddingDimension) {
		t.Fatalf("failed batch=%v", err)
	}
	stats, err := db.StatsMVCC()
	if err != nil || stats.CommitSeq != 0 {
		t.Fatalf("partial batch commit=%+v,%v", stats, err)
	}
	first, err := writer.PutCells(t.Context(), cmd, [][]float32{nil, {1, 0}})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := writer.PutCells(t.Context(), cmd, [][]float32{nil, nil})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Results) != 2 || first.Results[0].Coord != replay.Results[0].Coord || first.Results[1].Coord != replay.Results[1].Coord {
		t.Fatalf("replay coordinates changed: %+v / %+v", first, replay)
	}
	stats, err = db.StatsMVCC()
	if err != nil || stats.CommitSeq != 1 || stats.LogicalCells != 2 {
		t.Fatalf("duplicate batch commit=%+v,%v", stats, err)
	}
	cmd.Cells[1].RawContent = "different"
	if _, err := writer.PutCells(t.Context(), cmd, [][]float32{nil, nil}); !errors.Is(err, domain.ErrIdempotencyKeyConflict) {
		t.Fatalf("key conflict=%v", err)
	}
}
