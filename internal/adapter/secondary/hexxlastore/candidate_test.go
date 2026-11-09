package hexxlastore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

func TestCreateCandidatePreservesExistingRecoveryFiles(t *testing.T) {
	t.Parallel()
	for _, suffix := range []string{"", "-wal", "-changelog", ".rotate.state"} {
		dir := t.TempDir()
		root, err := os.OpenRoot(dir)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = root.Close() })
		const name = "target.hexxla"
		if err := root.WriteFile(name+suffix, []byte("existing"), 0o600); err != nil {
			t.Fatal(err)
		}
		err = CreateCandidate(t.Context(), filepath.Join(dir, name), nil, func(*hexxladb.DB) error {
			t.Fatal("populated a candidate for an occupied destination")
			return nil
		})
		if !errors.Is(err, os.ErrExist) {
			t.Fatalf("CreateCandidate = %v, want os.ErrExist", err)
		}
		got, err := root.ReadFile(name + suffix)
		if err != nil || string(got) != "existing" {
			t.Fatalf("original %q changed: %q, %v", suffix, got, err)
		}
	}
}

func TestCreateCandidatePublishesOnlyAfterPopulationSucceeds(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "target.hexxla")
	opts := &hexxladb.Options{EmbeddingDimension: 2}
	key, err := hexxladb.Pack(hexxladb.Coord{})
	if err != nil {
		t.Fatal(err)
	}
	populate := func(db *hexxladb.DB) error {
		return db.Update(func(tx *hexxladb.Tx) error {
			return tx.PutCell(ctx, hexxladb.NewFactCell(key, "candidate data", "seed", "fact", 1))
		})
	}
	failure := errors.New("embedding model unavailable")
	if err := CreateCandidate(ctx, path, opts, func(db *hexxladb.DB) error {
		if err := populate(db); err != nil {
			return err
		}
		return failure
	}); !errors.Is(err, failure) {
		t.Fatalf("failed candidate = %v, want %v", err, failure)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed candidate left artifacts: %v, %v", entries, err)
	}
	if err := CreateCandidate(ctx, path, opts, populate); err != nil {
		t.Fatal(err)
	}
	db, err := hexxladb.OpenExisting(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	assertCellContent(t, db, domain.AxialCoord{}, "candidate data")
}
