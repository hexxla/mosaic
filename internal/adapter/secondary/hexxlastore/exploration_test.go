package hexxlastore

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/hexxla/hexxladb"

	"github.com/hexxla/mosaic/internal/core/domain"
)

func TestGraphContextValidityAndHistoricalSearch(t *testing.T) {
	db, err := hexxladb.Open(filepath.Join(t.TempDir(), "graph.hexxla"), &hexxladb.Options{ChangelogEnabled: true, EmbeddingDimension: 2})
	if err != nil {
		t.Fatal(err)
	}
	live := NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })
	now := time.Now().UTC()
	future := now.Add(time.Hour).UnixNano()
	if err := db.Update(func(tx *hexxladb.Tx) error {
		for _, c := range []hexxladb.Coord{{}, {Q: 20}, {Q: 21}} {
			key, _ := hexxladb.Pack(c)
			rec := hexxladb.NewFactCell(key, "old", "fixture", "topic", 1)
			if c.Q == 21 {
				rec.Validity.ValidFrom = &future
			}
			if err := tx.PutCell(t.Context(), rec); err != nil {
				return err
			}
		}
		if err := tx.LinkCells(hexxladb.Coord{}, hexxladb.Coord{Q: 20}, "requires", 1, hexxladb.ProvenanceWire{}); err != nil {
			return err
		}
		return tx.LinkCells(hexxladb.Coord{Q: 20}, hexxladb.Coord{Q: 21}, "requires", 1, hexxladb.ProvenanceWire{})
	}); err != nil {
		t.Fatal(err)
	}
	cmd := &domain.LoadContextPackCommand{Seeds: []domain.AxialCoord{{}}, EdgeFilter: "requires", MaxHops: 2, AsOf: &now, IncludeEdges: true, MaxCells: 10}
	pack, err := NewContextPackAdapter(live).LoadFromSeeds(t.Context(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	if len(pack.Cells) != 2 || pack.Cells[1].Coord.Q != 20 || len(pack.Cells[0].Edges) != 1 {
		t.Fatalf("graph/validity mapping=%+v", pack)
	}
	if err := db.Update(func(tx *hexxladb.Tx) error {
		key, _ := hexxladb.Pack(hexxladb.Coord{})
		return tx.PutCell(t.Context(), hexxladb.NewFactCell(key, "new", "fixture", "topic", 1))
	}); err != nil {
		t.Fatal(err)
	}
	hits, err := NewCellReaderAdapter(live).SearchCells(t.Context(), &domain.CellSearchCommand{Query: "old", MaxResults: 10, Snapshot: &domain.SnapshotSelector{Seq: new(uint64(1))}}, nil)
	if err != nil || len(hits.Hits) != 3 {
		t.Fatalf("historical search=%+v,%v", hits, err)
	}
	changes, err := NewMemoryExplorerAdapter(live).ReadChanges(t.Context(), domain.ChangesQuery{MaxRecords: 2, Operations: []string{"delete_cell"}})
	if err != nil || len(changes.Events) != 0 || changes.NextSeq != 2 || !changes.MayHaveMore {
		t.Fatalf("filtered feed made no progress=%+v,%v", changes, err)
	}
}
