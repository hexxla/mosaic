package hexxlastore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/config"
	"github.com/sploitzberg/mosaic/internal/core/domain"
)

func TestOverwritePreservesNativeMetadata(t *testing.T) {
	db, err := hexxladb.Open(filepath.Join(t.TempDir(), "metadata.hexxla"), nil)
	if err != nil {
		t.Fatal(err)
	}
	live := NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })
	key, _ := hexxladb.Pack(hexxladb.Coord{})
	hint, _ := hexxladb.Pack(hexxladb.Coord{Q: 9})
	from, to := time.Now().Add(-time.Hour).UnixNano(), time.Now().Add(time.Hour).UnixNano()
	rec := hexxladb.CellRecord{Key: key, RawContent: "native", Tags: []string{"custom"}, ClusterHint: &hint, Validity: hexxladb.ValidityWire{ValidFrom: &from, ValidTo: &to}, Provenance: hexxladb.ProvenanceWire{CreatedAt: 1234}}
	if err := db.Update(func(tx *hexxladb.Tx) error { return tx.PutCell(t.Context(), rec) }); err != nil {
		t.Fatal(err)
	}
	writer := NewCellWriterAdapter(live, config.DeleteAutoMaintainConfig{})
	cmd := &domain.PutCellCommand{RawContent: "updated", SourceID: "source", AllowOverwrite: true, Metadata: &domain.CellMetadata{ExactTags: true}, Tags: []string{"custom"}}
	if _, err := writer.PutCell(t.Context(), cmd, nil); err != nil {
		t.Fatal(err)
	}
	if err := db.View(func(tx *hexxladb.Tx) error {
		got, _, err := tx.GetCell(key)
		if err != nil {
			return err
		}
		if got.Provenance.CreatedAt != 1234 || got.ClusterHint == nil || *got.ClusterHint != hint || got.Validity.ValidFrom == nil || *got.Validity.ValidFrom != from || *got.Validity.ValidTo != to || len(got.Tags) != 1 || got.Tags[0] != "custom" {
			t.Fatalf("lost metadata: %+v", got)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	lookup, err := NewCellReaderAdapter(live).GetCell(t.Context(), domain.AxialCoord{})
	if err != nil || lookup.Cell.ClusterHint == nil || lookup.Cell.ClusterHint.Q != 9 {
		t.Fatalf("lookup = %+v, %v", lookup, err)
	}
	cmd.Metadata = &domain.CellMetadata{Replace: true}
	if _, err := writer.PutCell(t.Context(), cmd, nil); err != nil {
		t.Fatal(err)
	}
	if err := db.View(func(tx *hexxladb.Tx) error {
		got, _, err := tx.GetCell(key)
		if got.Validity.ValidFrom != nil || got.ClusterHint != nil {
			t.Fatalf("replace did not clear metadata: %+v", got)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func TestFacetWriteRejectsStaleSource(t *testing.T) {
	db, err := hexxladb.Open(filepath.Join(t.TempDir(), "facet.hexxla"), nil)
	if err != nil {
		t.Fatal(err)
	}
	live := NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })
	key, _ := hexxladb.Pack(hexxladb.Coord{})
	if err := db.Update(func(tx *hexxladb.Tx) error {
		return tx.PutCell(t.Context(), hexxladb.CellRecord{Key: key, RawContent: "new"})
	}); err != nil {
		t.Fatal(err)
	}
	store := NewFacetEdgeStoreAdapter(live)
	hash := sha256.Sum256([]byte("old"))
	cmd := &domain.PutFacetCommand{DerivedContent: "derived", ExpectedContentSHA256: hex.EncodeToString(hash[:])}
	if err := store.PutFacet(t.Context(), cmd); !errors.Is(err, hexxladb.ErrFacetDerivationMismatch) {
		t.Fatalf("stale write = %v", err)
	}
	hash = sha256.Sum256([]byte("new"))
	cmd.ExpectedContentSHA256 = hex.EncodeToString(hash[:])
	if err := store.PutFacet(t.Context(), cmd); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetFacet(t.Context(), domain.AxialCoord{}, 0)
	if err != nil || !got.Found || got.DerivationHashHex != cmd.ExpectedContentSHA256 {
		t.Fatalf("facet = %+v, %v", got, err)
	}
}
