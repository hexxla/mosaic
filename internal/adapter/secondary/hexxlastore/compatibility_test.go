package hexxlastore

import (
	"bytes"
	"compress/gzip"
	"embed"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/config"
	"github.com/sploitzberg/mosaic/internal/core/domain"
)

//go:embed testdata/hexxladb-177f2a3.hexxla.gz testdata/hexxladb-177f2a3.hexxla-wal.gz
var compatibilityFixtures embed.FS

// The recovery-set fixtures were created by unmodified HexxlaDB 177f2a3 using
// its public API: one fact cell and a two-dimensional embedding in one Update.
// They exercise framed WAL/header checks and version-2 derived index metadata.
func TestCurrentHexxlaRecoverySetCompatibility(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	for _, suffix := range []string{"", "-wal"} {
		compressed, err := compatibilityFixtures.ReadFile("testdata/hexxladb-177f2a3.hexxla" + suffix + ".gz")
		if err != nil {
			t.Fatal(err)
		}
		reader, err := gzip.NewReader(bytes.NewReader(compressed))
		if err != nil {
			t.Fatal(err)
		}
		data, readErr := io.ReadAll(io.LimitReader(reader, (1<<20)+1))
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil || len(data) > 1<<20 {
			t.Fatalf("invalid compatibility fixture: read=%v close=%v bytes=%d", readErr, closeErr, len(data))
		}
		if err := root.WriteFile("compatibility.hexxla"+suffix, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, "compatibility.hexxla")
	db, err := hexxladb.OpenExisting(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	live := NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })
	hits, err := NewEmbeddingANNAdapter(live).Search(t.Context(), []float32{1, 0}, 1, 0)
	if err != nil || len(hits) != 1 || hits[0].RawContent != "current fixture" {
		t.Fatalf("current-format fixture search = %+v, %v", hits, err)
	}
	writer := NewCellWriterAdapter(live, config.DeleteAutoMaintainConfig{})
	if _, err := writer.PutCell(t.Context(), &domain.PutCellCommand{
		RawContent: "rewritten fixture", SourceID: "test", AllowOverwrite: true,
	}, nil); err != nil {
		t.Fatal(err)
	}
	if err := writer.PutEmbedding(t.Context(), domain.AxialCoord{}, []float32{0, 1}, ""); err != nil {
		t.Fatal(err)
	}
	if err := live.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := hexxladb.OpenExisting(path, nil)
	if err != nil {
		t.Fatalf("reopen after Mosaic writes: %v", err)
	}
	reopenedLive := NewLiveDB(reopened)
	t.Cleanup(func() { _ = reopenedLive.Close() })
	hits, err = NewEmbeddingANNAdapter(reopenedLive).Search(t.Context(), []float32{0, 1}, 1, 0)
	if err != nil || len(hits) != 1 || hits[0].RawContent != "rewritten fixture" {
		t.Fatalf("rewritten fixture search = %+v, %v", hits, err)
	}
}
