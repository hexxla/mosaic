package config_test

import (
	"testing"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/config"
)

func TestDefaultMosaicDatabaseLayout_matches_MosaicNewDatabaseOptions(t *testing.T) {
	t.Parallel()
	def := config.DefaultMosaicDatabaseLayout()
	o := config.NewMosaicDatabaseOptions(def)
	legacy := config.MosaicNewDatabaseOptions()
	if o.EnableMVCC != legacy.EnableMVCC ||
		o.PageSize != legacy.PageSize ||
		o.MaxValueBytes != legacy.MaxValueBytes ||
		o.EmbeddingDimension != legacy.EmbeddingDimension ||
		o.DistanceMetric != legacy.DistanceMetric {
		t.Fatalf("NewMosaicDatabaseOptions(Default) != MosaicNewDatabaseOptions: %+v vs %+v", o, legacy)
	}
	if def.PageSize != 4096 {
		t.Fatalf("default PageSize=%d want HexxlaDB validated profile 4096", def.PageSize)
	}
}

func TestParseDistanceMetricName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		in   string
		want hexxladb.DistanceMetric
	}{
		{"", hexxladb.DistanceCosine},
		{"cosine", hexxladb.DistanceCosine},
		{"L2", hexxladb.DistanceL2},
		{"euclidean", hexxladb.DistanceL2},
		{"dot", hexxladb.DistanceDotProduct},
	} {
		got, err := config.ParseDistanceMetricName(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.in, got, tc.want)
		}
	}
	if _, err := config.ParseDistanceMetricName("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseMosaicDatabaseLayoutFromCLI(t *testing.T) {
	t.Parallel()
	def := config.DefaultMosaicDatabaseLayout()
	got, err := config.ParseMosaicDatabaseLayoutFromCLI(def.EnableMVCC, uint(def.PageSize), uint(def.MaxValueBytes), uint(def.EmbeddingDimension), "cosine")
	if err != nil {
		t.Fatal(err)
	}
	if got != def {
		t.Fatalf("got %+v want %+v", got, def)
	}
}
