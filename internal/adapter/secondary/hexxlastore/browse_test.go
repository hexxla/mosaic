package hexxlastore

import (
	"bytes"
	"encoding/base64"
	"encoding/json/v2"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hexxla/hexxladb"

	"github.com/hexxla/mosaic/internal/core/domain"
)

func TestBrowseContinuationKeepsSnapshotAndDefersOversizedCells(t *testing.T) {
	db, err := hexxladb.Open(filepath.Join(t.TempDir(), "browse.hexxla"), &hexxladb.Options{MaxValueBytes: 16384})
	if err != nil {
		t.Fatal(err)
	}
	live := NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })
	store := NewMemoryExplorerAdapter(live)
	coords := hexxladb.WalkRings(nil, hexxladb.Coord{}, 8)
	if err := db.Update(func(tx *hexxladb.Tx) error {
		for i, c := range coords {
			key, _ := hexxladb.Pack(c)
			content := "original"
			if i == 0 {
				content = strings.Repeat("large", 1000)
			}
			if err := tx.PutCell(t.Context(), hexxladb.CellRecord{Key: key, RawContent: content}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	q := domain.BrowseQuery{Radius: 8, MaxCells: 7, MaxProbes: 10, MaxBytes: 1024, IncludeContent: true}
	seen := make(map[domain.AxialCoord]bool)
	var seq uint64
	deferred := 0
	for pageNum := 0; pageNum < 100; pageNum++ {
		page, err := store.Browse(t.Context(), q)
		if err != nil {
			t.Fatal(err)
		}
		if page.Probes > 10 {
			t.Fatalf("unbounded probes=%d", page.Probes)
		}
		if pageNum == 0 {
			seq = page.SnapshotSeq
			if err := db.Update(func(tx *hexxladb.Tx) error {
				key, _ := hexxladb.Pack(hexxladb.Coord{})
				return tx.PutCell(t.Context(), hexxladb.CellRecord{Key: key, RawContent: "changed"})
			}); err != nil {
				t.Fatal(err)
			}
		}
		if page.SnapshotSeq != seq {
			t.Fatal("snapshot changed")
		}
		for _, hit := range page.Cells {
			if seen[hit.Coord] {
				t.Fatalf("duplicate coord=%v", hit.Coord)
			}
			seen[hit.Coord] = true
			if hit.RawContent != "original" {
				t.Fatalf("mixed current content=%q", hit.RawContent)
			}
		}
		for _, c := range page.DeferredCells {
			if seen[c] {
				t.Fatal("duplicate deferred cell")
			}
			seen[c] = true
			deferred++
		}
		if page.Done {
			break
		}
		q.SnapshotSeq = new(seq)
		q.Ring = page.NextRing
		q.Offset = page.NextOffset
	}
	if len(seen) != len(coords) || deferred != 1 {
		t.Fatalf("seen=%d/%d deferred=%d", len(seen), len(coords), deferred)
	}
	chunkQuery := domain.CellChunkQuery{Snapshot: domain.SnapshotSelector{Seq: new(seq)}, Format: "record_json", MaxBytes: 113}
	var combined []byte
	for {
		chunk, err := store.ReadCellChunk(t.Context(), chunkQuery)
		if err != nil {
			t.Fatal(err)
		}
		data, err := base64.StdEncoding.DecodeString(chunk.DataBase64)
		if err != nil {
			t.Fatal(err)
		}
		combined = append(combined, data...)
		if chunk.Done {
			break
		}
		chunkQuery.Offset = chunk.NextOffset
	}
	var record hexxladb.CellRecord
	if err := json.Unmarshal(combined, &record); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal([]byte(record.RawContent), []byte(strings.Repeat("large", 1000))) {
		t.Fatal("record chunks mixed in updated snapshot")
	}
}
