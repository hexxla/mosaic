package config_test

import (
	"testing"

	"github.com/hexxla/hexxladb"

	"github.com/hexxla/mosaic/internal/config"
)

func TestMosaicDatabaseDefaultOptions(t *testing.T) {
	t.Setenv(config.EnvDBPassphrase, "")
	t.Setenv(config.EnvDBEncryptionKeyHex, "")
	loaded, err := config.ParseMosaicConfigYAML([]byte("version: 1\ndatabase:\n  passphrase: example-fixture\n"))
	if err != nil {
		t.Fatal(err)
	}
	o, err := config.BuildMosaicCreateOptions(config.DefaultMosaicDatabaseLayout(), config.HexxlaOpenParams{
		YAMLPassphrase: loaded.DatabasePassphrase,
	})
	if err != nil {
		t.Fatal(err)
	}
	if o.Passphrase != "example-fixture" {
		t.Fatal("creation options did not inherit the YAML passphrase")
	}
	if o.PageSize != 4096 || o.EmbeddingDimension != 384 || o.DistanceMetric != hexxladb.DistanceCosine {
		t.Fatalf("unexpected default layout: page size %d, embedding dimension %d, metric %v", o.PageSize, o.EmbeddingDimension, o.DistanceMetric)
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
	got, err := config.ParseMosaicDatabaseLayoutFromCLI(uint(def.PageSize), uint(def.MaxValueBytes), uint(def.EmbeddingDimension), "cosine")
	if err != nil {
		t.Fatal(err)
	}
	if got != def {
		t.Fatalf("got %+v want %+v", got, def)
	}
}
