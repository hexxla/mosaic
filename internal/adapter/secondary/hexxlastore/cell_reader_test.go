package hexxlastore

import (
	"path/filepath"
	"testing"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

func TestCellReaderReportsPartialAndCandidateLimits(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	db, err := hexxladb.Open(filepath.Join(t.TempDir(), "query.hexxla"), &hexxladb.Options{EmbeddingDimension: 2})
	if err != nil {
		t.Fatal(err)
	}
	live := NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })
	if err := db.Update(func(tx *hexxladb.Tx) error {
		for i := range 110 {
			key, err := hexxladb.Pack(hexxladb.Coord{Q: i})
			if err != nil {
				return err
			}
			rec := hexxladb.NewFactCell(key, "sample", "test", "fact", 1)
			vector := []float32{1, 0}
			if i == 109 {
				rec.Tags = append(rec.Tags, "rare")
				vector = []float32{0, 1}
			}
			if err := tx.PutCell(ctx, rec); err != nil {
				return err
			}
			if err := tx.PutEmbeddingWithOptions(key, vector, hexxladb.EmbeddingWriteOptions{DeferIndexMaintenance: true}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	reader := NewCellReaderAdapter(live)
	partial, err := reader.SearchCells(ctx, &domain.CellSearchCommand{MaxScanRows: 1}, nil)
	if err != nil || !partial.ScanLimitReached || len(partial.Hits) != 1 {
		t.Fatalf("partial query = %+v, %v", partial, err)
	}

	q := &domain.CellSearchCommand{RequireTags: []string{"rare"}, MaxResults: 1}
	capped, err := reader.SearchCells(ctx, q, []float32{1, 0})
	if err != nil || !capped.EmbeddingCandidateLimitReached || len(capped.Hits) != 0 {
		t.Fatalf("candidate-limited query = %+v, %v", capped, err)
	}

	q.MaxScanRows = 1
	exact, err := reader.SearchCells(ctx, q, []float32{1, 0})
	if err != nil || exact.ScanLimitReached || exact.EmbeddingCandidateLimitReached || exact.EmbeddingPath != "flat" || len(exact.Hits) != 1 {
		t.Fatalf("bounded exact query = %+v, %v", exact, err)
	}
}
