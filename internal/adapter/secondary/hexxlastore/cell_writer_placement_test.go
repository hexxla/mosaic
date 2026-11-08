package hexxlastore

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/config"
	"github.com/sploitzberg/mosaic/internal/core/domain"
)

func TestCellWriterAdapter_exact_and_near_anchor_placement(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "placement.hexxla")
	opts := &hexxladb.Options{EnableMVCC: true, PageSize: 4096}
	db, err := hexxladb.Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	live := NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })
	writer := NewCellWriterAdapter(live, config.DeleteAutoMaintainConfig{})

	base := domain.PutCellCommand{
		Coord:      domain.AxialCoord{Q: 2, R: -1},
		RawContent: "first",
		SourceID:   "test",
		Confidence: 0.9,
		Kind:       domain.CellPutKindFact,
		Placement:  domain.CellPlacementExact,
	}
	placed, err := writer.PutCell(t.Context(), &base)
	if err != nil {
		t.Fatal(err)
	}
	if !placed.OK || placed.Coord != base.Coord || placed.Placement != domain.CellPlacementExact || placed.Replaced {
		t.Fatalf("first exact result: %+v", placed)
	}

	conflict := base
	conflict.RawContent = "must not replace"
	if _, err := writer.PutCell(t.Context(), &conflict); !errors.Is(err, domain.ErrCellCoordinateOccupied) {
		t.Fatalf("occupied exact error=%v", err)
	}
	assertCellContent(t, db, base.Coord, "first")

	overwrite := base
	overwrite.RawContent = "replacement"
	overwrite.AllowOverwrite = true
	replaced, err := writer.PutCell(t.Context(), &overwrite)
	if err != nil {
		t.Fatal(err)
	}
	if !replaced.Replaced || replaced.Coord != base.Coord {
		t.Fatalf("overwrite result: %+v", replaced)
	}
	assertCellContent(t, db, base.Coord, "replacement")

	near := base
	near.RawContent = "neighbor"
	near.Placement = domain.CellPlacementNearAnchor
	near.MaxRadius = 1
	neighbor, err := writer.PutCell(t.Context(), &near)
	if err != nil {
		t.Fatal(err)
	}
	if neighbor.Placement != domain.CellPlacementNearAnchor || neighbor.Replaced || neighbor.Probes != 1 {
		t.Fatalf("near-anchor result: %+v", neighbor)
	}
	actual := hexxladb.Coord{Q: neighbor.Coord.Q, R: neighbor.Coord.R}
	anchor := hexxladb.Coord{Q: base.Coord.Q, R: base.Coord.R}
	if actual.Distance(anchor) != 1 {
		t.Fatalf("placed coordinate %+v is not adjacent to anchor %+v", actual, anchor)
	}
	assertCellContent(t, db, neighbor.Coord, "neighbor")
}

func assertCellContent(t *testing.T, db *hexxladb.DB, coord domain.AxialCoord, want string) {
	t.Helper()
	key, err := hexxladb.Pack(hexxladb.Coord{Q: coord.Q, R: coord.R})
	if err != nil {
		t.Fatal(err)
	}
	err = db.View(func(tx *hexxladb.Tx) error {
		rec, ok, err := tx.GetCell(key)
		if err != nil {
			return err
		}
		if !ok {
			t.Fatalf("cell at %+v not found", coord)
		}
		if rec.RawContent != want {
			t.Fatalf("cell at %+v content=%q want %q", coord, rec.RawContent, want)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCellWriterReportsCommittedDeleteWhenPruningFails(t *testing.T) {
	t.Parallel()
	db, err := hexxladb.Open(filepath.Join(t.TempDir(), "delete.hexxla"), &hexxladb.Options{
		EnableMVCC:    true,
		MVCCRetention: hexxladb.MVCCRetention{RetainCommitsBehindHead: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	live := NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })
	maintenance := config.DeleteAutoMaintainConfig{Enabled: true, Prune: true, MVCCPruneProfile: "invalid-profile"}
	if err := runPostDeleteMaintain(t.Context(), live, maintenance); !errors.Is(err, hexxladb.ErrInvalidArgument) {
		t.Fatalf("test configuration did not produce a pruning error: %v", err)
	}
	writer := NewCellWriterAdapter(live, maintenance)
	if _, err := writer.PutCell(t.Context(), &domain.PutCellCommand{RawContent: "test", SourceID: "test"}); err != nil {
		t.Fatal(err)
	}
	removed, err := writer.DeleteCell(t.Context(), &domain.DeleteCellCommand{})
	if err != nil || !removed {
		t.Fatalf("delete returned removed=%t, err=%v", removed, err)
	}
	removed, err = writer.DeleteCell(t.Context(), &domain.DeleteCellCommand{})
	if err != nil || removed {
		t.Fatalf("repeat delete returned removed=%t, err=%v", removed, err)
	}
}
