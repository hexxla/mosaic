//go:build ignore

// Run from the repository root to regenerate the compatibility recovery set.
package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/hexxla/hexxladb"
)

const (
	fixtureVersion = "v0.7.1-0.20261109154236-177f2a3854b9"
	fixtureDir     = "internal/adapter/secondary/hexxlastore/testdata"
)

func main() {
	if err := generate(); err != nil {
		slog.Error("generate compatibility recovery set", "err", err)
		os.Exit(1)
	}
}

func generate() error {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fmt.Errorf("missing build information")
	}
	matched := false
	for _, dep := range info.Deps {
		if dep.Path == "github.com/hexxla/hexxladb" {
			matched = dep.Version == fixtureVersion && dep.Replace == nil
		}
	}
	if !matched {
		return fmt.Errorf("generator requires unmodified HexxlaDB %s", fixtureVersion)
	}
	temp, err := os.MkdirTemp("", "mosaic-compatibility-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(temp) }()
	path := filepath.Join(temp, "fixture.hexxla")
	db, err := hexxladb.Open(path, &hexxladb.Options{EmbeddingDimension: 2})
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	key, err := hexxladb.Pack(hexxladb.Coord{})
	if err != nil {
		return err
	}
	if err := db.Update(func(tx *hexxladb.Tx) error {
		if err := tx.PutCell(context.Background(), hexxladb.NewFactCell(key, "current fixture", "fixture", "fact", 1)); err != nil {
			return err
		}
		return tx.PutEmbedding(key, []float32{1, 0})
	}); err != nil {
		return err
	}
	if err := db.Close(); err != nil {
		return err
	}
	output, err := os.OpenRoot(fixtureDir)
	if err != nil {
		return err
	}
	defer func() { _ = output.Close() }()
	for _, suffix := range []string{"", "-wal"} {
		stat, err := os.Stat(path + suffix)
		if err != nil {
			return err
		}
		if stat.Size() > 1<<20 {
			return fmt.Errorf("fixture exceeds 1 MiB")
		}
		data, err := os.ReadFile(path + suffix)
		if err != nil {
			return err
		}
		var compressed bytes.Buffer
		writer := gzip.NewWriter(&compressed)
		if _, err := writer.Write(data); err != nil {
			return err
		}
		if err := writer.Close(); err != nil {
			return err
		}
		if err := output.WriteFile("hexxladb-177f2a3.hexxla"+suffix+".gz", compressed.Bytes(), 0o600); err != nil {
			return err
		}
	}
	return nil
}
