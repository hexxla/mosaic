package hexxlastore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/hexxla/hexxladb"

	"github.com/hexxla/mosaic/internal/core/domain"
)

func TestEmbeddingModelSameDimensionSwapRequiresReembed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.hexxla")
	db, err := hexxladb.Open(path, &hexxladb.Options{EmbeddingDimension: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	const oldModel, newModel = "old-model:v1", "new-model:v1"
	if err := EnsureEmbeddingModel(db, oldModel); err != nil {
		t.Fatal(err)
	}
	for q := range 2 {
		writeModelTestCell(t, db, q)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = hexxladb.OpenExisting(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureEmbeddingModel(db, newModel); !errors.Is(err, ErrEmbeddingModelMismatch) {
		t.Fatalf("same-width model change = %v, want model mismatch", err)
	}

	newVector := []float32{0, 1}
	modelError := errors.New("model unavailable")
	var calls int
	failing := atomicCellEmbedder(func(context.Context, string, int) ([]float32, error) {
		calls++
		if calls == 2 {
			return nil, modelError
		}
		return newVector, nil
	})
	if err := Reembed(t.Context(), db, newModel, failing); !errors.Is(err, modelError) {
		t.Fatalf("interrupted re-embedding = %v, want model error", err)
	}
	if err := EnsureEmbeddingModel(db, oldModel); err != nil {
		t.Fatalf("old identity after rollback: %v", err)
	}
	for q := range 2 {
		assertAtomicCellAndEmbedding(t, db, domain.AxialCoord{Q: q}, "cell content", []float32{1, 0}, true)
	}
	replacement := atomicCellEmbedder(func(_ context.Context, text string, dim int) ([]float32, error) {
		if text != "cell content" || dim != len(newVector) {
			t.Fatalf("embed input = %q, dimension %d", text, dim)
		}
		return newVector, nil
	})
	if err := Reembed(t.Context(), db, newModel, replacement); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = hexxladb.OpenExisting(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureEmbeddingModel(db, newModel); err != nil {
		t.Fatalf("new identity after reopen: %v", err)
	}
	if err := EnsureEmbeddingModel(db, oldModel); !errors.Is(err, ErrEmbeddingModelMismatch) {
		t.Fatalf("old model after re-embedding = %v, want model mismatch", err)
	}
	for q := range 2 {
		assertAtomicCellAndEmbedding(t, db, domain.AxialCoord{Q: q}, "cell content", newVector, true)
	}
}

func TestEmbeddingModelRejectsUnidentifiedVectors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.hexxla")
	db, err := hexxladb.Open(path, &hexxladb.Options{EmbeddingDimension: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	writeModelTestCell(t, db, 0)
	if err := EnsureEmbeddingModel(db, "new-model"); !errors.Is(err, ErrEmbeddingModelMismatch) {
		t.Fatalf("unidentified vectors = %v, want model mismatch", err)
	}
	replacement := atomicCellEmbedder(func(context.Context, string, int) ([]float32, error) {
		return []float32{0, 1}, nil
	})
	if err := Reembed(t.Context(), db, "new-model", replacement); err != nil {
		t.Fatal(err)
	}
	if err := EnsureEmbeddingModel(db, "new-model"); err != nil {
		t.Fatal(err)
	}
}

func writeModelTestCell(t *testing.T, db *hexxladb.DB, q int) {
	t.Helper()
	key, err := hexxladb.Pack(hexxladb.Coord{Q: q})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Update(func(tx *hexxladb.Tx) error {
		if err := tx.PutCell(t.Context(), hexxladb.NewFactCell(key, "cell content", "test", "test", 1)); err != nil {
			return err
		}
		return tx.PutEmbedding(key, []float32{1, 0})
	}); err != nil {
		t.Fatal(err)
	}
}
