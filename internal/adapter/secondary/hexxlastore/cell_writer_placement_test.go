package hexxlastore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/config"
	"github.com/sploitzberg/mosaic/internal/core/domain"
)

func TestPutEmbeddingChecksExpectedCellContentAtomically(t *testing.T) {
	t.Parallel()
	db, live := openLifecycleDB(t)
	ctx := t.Context()
	writer := NewCellWriterAdapter(live, config.DeleteAutoMaintainConfig{})
	cmd := domain.PutCellCommand{RawContent: "current content", SourceID: "test"}
	if _, err := writer.PutCell(ctx, &cmd, nil); err != nil {
		t.Fatal(err)
	}
	if err := writer.PutEmbedding(ctx, cmd.Coord, []float32{1, 0}, ""); err != nil {
		t.Fatal(err)
	}
	stale := sha256.Sum256([]byte("previous content"))
	if err := writer.PutEmbedding(ctx, cmd.Coord, []float32{0, 1}, hex.EncodeToString(stale[:])); !errors.Is(err, domain.ErrCellContentChanged) {
		t.Fatalf("stale embedding error = %v", err)
	}
	key, err := hexxladb.Pack(hexxladb.Coord{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.View(func(tx *hexxladb.Tx) error {
		vec, found, err := tx.GetEmbedding(key)
		if err == nil && (!found || !slices.Equal(vec, []float32{1, 0})) {
			t.Errorf("stale write changed embedding: %v, found=%t", vec, found)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	current := sha256.Sum256([]byte(cmd.RawContent))
	if err := writer.PutEmbedding(ctx, cmd.Coord, []float32{0, 1}, hex.EncodeToString(current[:])); err != nil {
		t.Fatalf("matching content rejected: %v", err)
	}
}

func TestCellWriterAdapter_exact_and_near_anchor_placement(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "placement.hexxla")
	opts := &hexxladb.Options{PageSize: 4096}
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
	placed, err := writer.PutCell(t.Context(), &base, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !placed.OK || placed.Coord != base.Coord || placed.Placement != domain.CellPlacementExact || placed.Replaced {
		t.Fatalf("first exact result: %+v", placed)
	}

	conflict := base
	conflict.RawContent = "must not replace"
	if _, err := writer.PutCell(t.Context(), &conflict, nil); !errors.Is(err, domain.ErrCellCoordinateOccupied) {
		t.Fatalf("occupied exact error=%v", err)
	}
	assertCellContent(t, db, base.Coord, "first")

	overwrite := base
	overwrite.RawContent = "replacement"
	overwrite.AllowOverwrite = true
	replaced, err := writer.PutCell(t.Context(), &overwrite, nil)
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
	near.IdempotencyKey = "neighbor"
	near.MaxRadius = 1
	neighbor, err := writer.PutCell(t.Context(), &near, nil)
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
	before := atomicCommitSeq(t, db)
	if replay, err := writer.PutCell(t.Context(), &near, nil); err != nil || replay != neighbor {
		t.Fatalf("transactional replay = %+v, %v; want %+v", replay, err, neighbor)
	}
	near.RawContent = "different request"
	if _, err := writer.PutCell(t.Context(), &near, nil); !errors.Is(err, domain.ErrIdempotencyKeyConflict) {
		t.Fatalf("reused key = %v, want conflict", err)
	}
	if got := atomicCommitSeq(t, db); got != before {
		t.Fatalf("replay or conflict advanced commit sequence from %d to %d", before, got)
	}
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
	if _, err := writer.PutCell(t.Context(), &domain.PutCellCommand{RawContent: "test", SourceID: "test"}, nil); err != nil {
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

func TestCellWriterProtectsVectorAndSeamIdentity(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	db, err := hexxladb.Open(filepath.Join(t.TempDir(), "identity.hexxla"), &hexxladb.Options{EmbeddingDimension: 2})
	if err != nil {
		t.Fatal(err)
	}
	live := NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })
	writer := NewCellWriterAdapter(live, config.DeleteAutoMaintainConfig{})
	a, b := domain.AxialCoord{Q: 1}, domain.AxialCoord{Q: 4}
	if err := writer.PutEmbedding(ctx, a, []float32{1, 0}, ""); !errors.Is(err, hexxladb.ErrCellNotFound) {
		t.Fatalf("embedding at vacant coordinate = %v", err)
	}
	key, err := hexxladb.Pack(hexxladb.Coord{Q: a.Q, R: a.R})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Update(func(tx *hexxladb.Tx) error {
		return tx.PutEmbedding(key, []float32{1, 0}) // Orphan left by an older Mosaic writer.
	}); err != nil {
		t.Fatal(err)
	}
	cmd := domain.PutCellCommand{Coord: a, RawContent: "original", SourceID: "test"}
	if _, err := writer.PutCell(ctx, &cmd, nil); err != nil {
		t.Fatal(err)
	}
	if err := db.View(func(tx *hexxladb.Tx) error {
		_, found, err := tx.GetEmbedding(key)
		if found {
			t.Error("new cell inherited an orphan embedding")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := writer.PutEmbedding(ctx, a, []float32{1, 0}, ""); err != nil {
		t.Fatal(err)
	}
	cmd.RawContent = "replacement"
	cmd.AllowOverwrite = true
	if _, err := writer.PutCell(ctx, &cmd, nil); err != nil {
		t.Fatal(err)
	}
	matches, err := NewEmbeddingANNAdapter(live).Search(ctx, []float32{1, 0}, 1, 0)
	if err != nil || len(matches) != 0 {
		t.Fatalf("old vector still ranks replacement: %v, %v", matches, err)
	}

	if _, err := writer.PutCell(ctx, &domain.PutCellCommand{Coord: b, RawContent: "peer", SourceID: "test"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := db.Update(func(tx *hexxladb.Tx) error {
		return tx.MarkConflict(hexxladb.Coord{Q: a.Q}, hexxladb.Coord{Q: b.Q}, "original conflict")
	}); err != nil {
		t.Fatal(err)
	}
	if removed, err := writer.DeleteCell(ctx, &domain.DeleteCellCommand{Coord: a}); err != nil || !removed {
		t.Fatalf("delete = %t, %v", removed, err)
	}
	if _, err := writer.PutCell(ctx, &cmd, nil); !errors.Is(err, domain.ErrCellCoordinateOccupied) {
		t.Fatalf("exact reuse of tombstone = %v", err)
	}
	cmd.AllowOverwrite = false
	cmd.Placement = domain.CellPlacementNearAnchor
	cmd.IdempotencyKey = "skip-tombstone"
	cmd.MaxRadius = 1
	placed, err := writer.PutCell(ctx, &cmd, nil)
	if err != nil || placed.Coord == a {
		t.Fatalf("near-anchor placement = %+v, %v", placed, err)
	}
}
