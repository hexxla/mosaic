package hexxlastore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/config"
	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/services"
)

type atomicCellEmbedder func(context.Context, string, int) ([]float32, error)

func (e atomicCellEmbedder) Embed(ctx context.Context, text string, dimension int) ([]float32, error) {
	return e(ctx, text, dimension)
}

func TestPutCellWithEmbeddingPublishesOneCommitAtSelectedCoordinate(t *testing.T) {
	t.Parallel()
	for _, placement := range []domain.CellPlacementMode{domain.CellPlacementExact, domain.CellPlacementNearAnchor} {
		t.Run(string(placement), func(t *testing.T) {
			t.Parallel()
			db, live := openLifecycleDB(t)
			writer := NewCellWriterAdapter(live, config.DeleteAutoMaintainConfig{})
			anchor := domain.AxialCoord{Q: 10, R: -3}
			var idempotencyKey string
			if placement == domain.CellPlacementNearAnchor {
				idempotencyKey = "atomic-placement"
				if _, err := writer.PutCell(t.Context(), &domain.PutCellCommand{Coord: anchor, RawContent: "anchor", SourceID: "test"}, nil); err != nil {
					t.Fatal(err)
				}
			}
			vector := []float32{0.25, 0.75}
			embedder := atomicCellEmbedder(func(_ context.Context, text string, dimension int) ([]float32, error) {
				if text != "embedded content" || dimension != 2 {
					t.Fatalf("embedding input = %q, dimension=%d", text, dimension)
				}
				return slices.Clone(vector), nil
			})
			service := services.NewCellMutationService(writer, embedder, 2)
			before := atomicCommitSeq(t, db)
			out, err := service.PutCell(t.Context(), &domain.PutCellCommand{
				Coord: anchor, RawContent: "embedded content", SourceID: "test", Placement: placement, Embed: true, IdempotencyKey: idempotencyKey,
			})
			if err != nil || !out.OK || !out.Embedded || out.Replaced || out.Placement != placement {
				t.Fatalf("atomic put = %+v, %v", out, err)
			}
			if placement == domain.CellPlacementExact && out.Coord != anchor {
				t.Fatalf("exact coordinate changed: %+v", out)
			}
			if placement == domain.CellPlacementNearAnchor && out.Coord == anchor {
				t.Fatal("near-anchor put overwrote its occupied anchor")
			}
			if got := atomicCommitSeq(t, db); got != before+1 {
				t.Fatalf("cell and embedding used %d commits, want 1", got-before)
			}
			assertAtomicCellAndEmbedding(t, db, out.Coord, "embedded content", vector, true)
			if placement == domain.CellPlacementNearAnchor {
				assertAtomicCellAndEmbedding(t, db, anchor, "anchor", nil, true)
			}
		})
	}
}

func TestPutCellEmbeddingFailuresRollbackNewAndReplacedCells(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		vector []float32
		want   error
	}{
		{name: "wrong dimension", vector: []float32{1}, want: hexxladb.ErrEmbeddingDimension},
		{name: "NaN", vector: []float32{float32(math.NaN()), 1}, want: hexxladb.ErrInvalidArgument},
		{name: "infinite", vector: []float32{1, float32(math.Inf(1))}, want: hexxladb.ErrInvalidArgument},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			db, live := openLifecycleDB(t)
			writer := NewCellWriterAdapter(live, config.DeleteAutoMaintainConfig{})
			original := domain.PutCellCommand{Coord: domain.AxialCoord{Q: 1}, RawContent: "original", SourceID: "test", Embed: true}
			oldVector := []float32{1, 0}
			if _, err := writer.PutCell(t.Context(), &original, oldVector); err != nil {
				t.Fatal(err)
			}
			for _, overwrite := range []bool{false, true} {
				cmd := original
				cmd.RawContent, cmd.AllowOverwrite = "must roll back", overwrite
				if !overwrite {
					cmd.Coord = domain.AxialCoord{Q: 4}
				}
				before := atomicCommitSeq(t, db)
				out, err := writer.PutCell(t.Context(), &cmd, test.vector)
				if !errors.Is(err, test.want) || out != (domain.PutCellMutationResult{}) {
					t.Fatalf("overwrite=%t: put = %+v, %v; want %v", overwrite, out, err, test.want)
				}
				if after := atomicCommitSeq(t, db); after != before {
					t.Fatalf("failed put committed sequence %d after %d", after, before)
				}
				assertAtomicCellAndEmbedding(t, db, domain.AxialCoord{Q: 4}, "", nil, false)
				assertAtomicCellAndEmbedding(t, db, original.Coord, original.RawContent, oldVector, true)
			}
		})
	}
}

func TestPutCellDatabaseFailuresRestoreInvalidatedVector(t *testing.T) {
	t.Parallel()
	for _, failure := range []string{"encoded value limit", "post-stage rejection"} {
		t.Run(failure, func(t *testing.T) {
			t.Parallel()
			injected := errors.New("cell write rejected after staging")
			reject := false
			opts := &hexxladb.Options{EmbeddingDimension: 2, MaxValueBytes: 512}
			if failure == "post-stage rejection" {
				opts.AfterPutCell = hexxladb.AfterPutCellHookFunc(func(context.Context, hexxladb.CellRecord) error {
					if reject {
						return injected
					}
					return nil
				})
			}
			db, err := hexxladb.Open(filepath.Join(t.TempDir(), "atomic.hexxla"), opts)
			if err != nil {
				t.Fatal(err)
			}
			live := NewLiveDB(db)
			t.Cleanup(func() { _ = live.Close() })
			writer := NewCellWriterAdapter(live, config.DeleteAutoMaintainConfig{})
			cmd := domain.PutCellCommand{Coord: domain.AxialCoord{Q: 1}, RawContent: "original", SourceID: "test", Embed: true}
			if _, err := writer.PutCell(t.Context(), &cmd, []float32{1, 0}); err != nil {
				t.Fatal(err)
			}
			before := atomicCommitSeq(t, db)
			reject = true
			cmd.RawContent, cmd.AllowOverwrite = "replacement", true
			if failure == "encoded value limit" {
				cmd.RawContent = strings.Repeat("oversized record", 100)
			}
			out, err := writer.PutCell(t.Context(), &cmd, []float32{0, 1})
			if err == nil || out != (domain.PutCellMutationResult{}) {
				t.Fatalf("database rejection returned %+v, %v", out, err)
			}
			if failure == "post-stage rejection" && !errors.Is(err, injected) {
				t.Fatalf("lost staged-write error cause: %v", err)
			}
			if after := atomicCommitSeq(t, db); after != before {
				t.Fatalf("failed replacement committed sequence %d after %d", after, before)
			}
			assertAtomicCellAndEmbedding(t, db, cmd.Coord, "original", []float32{1, 0}, true)
		})
	}
}

func TestAtomicOverwritePreservesCoordinateAndContentHashGuards(t *testing.T) {
	t.Parallel()
	db, live := openLifecycleDB(t)
	writer := NewCellWriterAdapter(live, config.DeleteAutoMaintainConfig{})
	cmd := domain.PutCellCommand{Coord: domain.AxialCoord{Q: 8}, RawContent: "original", SourceID: "test", Embed: true}
	if _, err := writer.PutCell(t.Context(), &cmd, []float32{1, 0}); err != nil {
		t.Fatal(err)
	}
	before := atomicCommitSeq(t, db)
	cmd.RawContent = "replacement"
	if _, err := writer.PutCell(t.Context(), &cmd, []float32{0, 1}); !errors.Is(err, domain.ErrCellCoordinateOccupied) {
		t.Fatalf("overwrite without permission = %v", err)
	}
	if atomicCommitSeq(t, db) != before {
		t.Fatal("denied overwrite committed")
	}
	assertAtomicCellAndEmbedding(t, db, cmd.Coord, "original", []float32{1, 0}, true)
	cmd.AllowOverwrite = true
	out, err := writer.PutCell(t.Context(), &cmd, []float32{0, 1})
	if err != nil || !out.OK || !out.Embedded || !out.Replaced || out.Coord != cmd.Coord || atomicCommitSeq(t, db) != before+1 {
		t.Fatalf("allowed atomic replacement = %+v, %v", out, err)
	}
	assertAtomicCellAndEmbedding(t, db, cmd.Coord, "replacement", []float32{0, 1}, true)
	stale := sha256.Sum256([]byte("original"))
	if err := writer.PutEmbedding(t.Context(), cmd.Coord, []float32{1, 0}, hex.EncodeToString(stale[:])); !errors.Is(err, domain.ErrCellContentChanged) {
		t.Fatalf("stale content guard accepted after atomic overwrite: %v", err)
	}
	assertAtomicCellAndEmbedding(t, db, cmd.Coord, "replacement", []float32{0, 1}, true)
}

func atomicCommitSeq(t *testing.T, db *hexxladb.DB) uint64 {
	t.Helper()
	stats, err := db.StatsMVCC()
	if err != nil {
		t.Fatal(err)
	}
	return stats.CommitSeq
}

func assertAtomicCellAndEmbedding(t *testing.T, db *hexxladb.DB, coord domain.AxialCoord, content string, vector []float32, found bool) {
	t.Helper()
	key, err := hexxladb.Pack(hexxladb.Coord{Q: coord.Q, R: coord.R})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.View(func(tx *hexxladb.Tx) error {
		rec, cellFound, err := tx.GetCell(key)
		if err != nil {
			return err
		}
		if cellFound != found || (found && rec.RawContent != content) {
			t.Errorf("cell at %+v = %q, found=%t; want %q, found=%t", coord, rec.RawContent, cellFound, content, found)
		}
		got, vectorFound, err := tx.GetEmbedding(key)
		if err != nil {
			return err
		}
		if vectorFound != (len(vector) > 0) || !slices.Equal(got, vector) {
			t.Errorf("embedding at %+v = %v, found=%t; want %v", coord, got, vectorFound, vector)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
