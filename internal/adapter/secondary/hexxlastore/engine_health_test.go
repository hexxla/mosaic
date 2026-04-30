package hexxlastore

import (
	"path/filepath"
	"testing"

	"github.com/hexxla/hexxladb"
)

func TestEngineHealthAdapter_Check_includes_layout_and_integrity(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "health.hexxla")
	db, err := hexxladb.Open(path, &hexxladb.Options{
		EnableMVCC:         true,
		EmbeddingDimension: 384,
		DistanceMetric:     hexxladb.DistanceCosine,
		PageSize:           4096,
	})
	if err != nil {
		t.Fatal(err)
	}
	live := NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })
	ad := NewEngineHealthAdapter(live, path)
	summary, err := ad.Check(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if summary.Disk.PrimaryPath != path {
		t.Fatalf("Disk.PrimaryPath: got %q want %q", summary.Disk.PrimaryPath, path)
	}
	if summary.Disk.PrimaryBytes <= 0 {
		t.Fatalf("Disk.PrimaryBytes: want > 0 for new DB file")
	}
	if summary.Disk.TotalBytes != summary.Disk.PrimaryBytes+summary.Disk.WALBytes {
		t.Fatalf("Disk.TotalBytes inconsistent")
	}
	if summary.DatabaseLayout.PageSize != db.PageSize() {
		t.Fatalf("PageSize: got %d want %d", summary.DatabaseLayout.PageSize, db.PageSize())
	}
	if summary.DatabaseLayout.MaxValueBytes != db.MaxValueBytes() {
		t.Fatalf("MaxValueBytes: got %d want %d", summary.DatabaseLayout.MaxValueBytes, db.MaxValueBytes())
	}
	if summary.DatabaseLayout.EmbeddingDimension != 384 {
		t.Fatalf("EmbeddingDimension: got %d", summary.DatabaseLayout.EmbeddingDimension)
	}
	if summary.DatabaseLayout.EmbeddingMetric != "cosine" {
		t.Fatalf("EmbeddingMetric: got %q", summary.DatabaseLayout.EmbeddingMetric)
	}
	if !summary.IntegrityOK {
		t.Fatal("expected IntegrityOK on empty healthy DB")
	}
}

func Test_embeddingMetricLabel_no_embeddings(t *testing.T) {
	t.Parallel()
	if got := embeddingMetricLabel(0, hexxladb.DistanceCosine); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}
