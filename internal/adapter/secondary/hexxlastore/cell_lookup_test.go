package hexxlastore

import (
	"slices"
	"testing"

	"github.com/hexxla/hexxladb"

	"github.com/hexxla/mosaic/internal/config"
	"github.com/hexxla/mosaic/internal/core/domain"
	"github.com/hexxla/mosaic/internal/core/services"
)

func TestCellLookupAndDefaultLexicalSearchReachFarCoordinate(t *testing.T) {
	t.Parallel()
	_, live := openLifecycleDB(t)
	writer := NewCellWriterAdapter(live, config.DeleteAutoMaintainConfig{})
	coord := domain.AxialCoord{Q: 100, R: -2}
	cmd := &domain.PutCellCommand{
		Coord: coord, RawContent: "distant exact memory", SourceID: "source-identity", Tags: []string{"distant"}, Confidence: 0.875,
	}
	if _, err := writer.PutCell(t.Context(), cmd, nil); err != nil {
		t.Fatal(err)
	}
	reader := NewCellReaderAdapter(live)
	lookup, err := reader.GetCell(t.Context(), coord)
	if err != nil || !lookup.Found || lookup.Cell == nil {
		t.Fatalf("far coordinate lookup = %+v, %v", lookup, err)
	}
	if lookup.Cell.Coord != coord || lookup.Cell.RawContent != cmd.RawContent || lookup.Cell.SourceID != cmd.SourceID ||
		lookup.Cell.Confidence != cmd.Confidence || !slices.Contains(lookup.Cell.Tags, "distant") || lookup.Cell.Score != 0 {
		t.Fatalf("exact lookup projection = %+v", lookup.Cell)
	}
	// Canonical Radius=0 has unrestricted spatial scope. Neither the adapter
	// nor the application's default lexical search may restore a 32-ring cap.
	q := &domain.CellSearchCommand{Query: "distant", Mode: domain.SearchLexical}
	hits, err := reader.SearchCells(t.Context(), q, nil)
	if err != nil || len(hits.Hits) != 1 || hits.Hits[0].Coord != coord {
		t.Fatalf("unrestricted adapter search = %+v, %v", hits, err)
	}
	service := services.NewCellRetrievalService(reader, nil, nil, 2)
	hits, err = service.SearchCells(t.Context(), &domain.CellSearchCommand{Query: "distant"})
	if err != nil || hits.Mode != domain.SearchLexical || len(hits.Hits) != 1 || hits.Hits[0].Coord != coord {
		t.Fatalf("default lexical search = %+v, %v", hits, err)
	}
	lookup.Cell.RawContent = "caller changed this snapshot"
	lookup.Cell.Tags[0] = "caller tag"
	fresh, err := reader.GetCell(t.Context(), coord)
	if err != nil || fresh.Cell == nil || fresh.Cell.RawContent != cmd.RawContent || slices.Contains(fresh.Cell.Tags, "caller tag") {
		t.Fatalf("lookup exposed mutable storage state: %+v, %v", fresh, err)
	}
}

func TestCellLookupDistinguishesMissingEmptyAndTombstonedCells(t *testing.T) {
	t.Parallel()
	db, live := openLifecycleDB(t)
	reader := NewCellReaderAdapter(live)
	coord := domain.AxialCoord{Q: 100}
	missing, err := reader.GetCell(t.Context(), coord)
	if err != nil || missing.Found || missing.Cell != nil {
		t.Fatalf("missing lookup = %+v, %v", missing, err)
	}
	key, err := hexxladb.Pack(hexxladb.Coord{Q: coord.Q, R: coord.R})
	if err != nil {
		t.Fatal(err)
	}
	// Native empty records are still present identities even though the
	// Mosaic write service requires nonempty content for its own writes.
	if err := db.Update(func(tx *hexxladb.Tx) error {
		return tx.PutCell(t.Context(), hexxladb.CellRecord{Key: key})
	}); err != nil {
		t.Fatal(err)
	}
	empty, err := reader.GetCell(t.Context(), coord)
	if err != nil || !empty.Found || empty.Cell == nil || empty.Cell.Coord != coord || empty.Cell.RawContent != "" {
		t.Fatalf("empty-content lookup = %+v, %v", empty, err)
	}
	writer := NewCellWriterAdapter(live, config.DeleteAutoMaintainConfig{})
	removed, err := writer.DeleteCell(t.Context(), &domain.DeleteCellCommand{Coord: coord})
	if err != nil || !removed {
		t.Fatalf("delete empty cell = %t, %v", removed, err)
	}
	tombstone, err := reader.GetCell(t.Context(), coord)
	if err != nil || tombstone.Found || tombstone.Cell != nil {
		t.Fatalf("tombstoned lookup = %+v, %v", tombstone, err)
	}
}
