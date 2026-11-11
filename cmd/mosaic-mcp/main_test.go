package main

import (
	"errors"
	"flag"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hexxla/hexxladb"

	"github.com/hexxla/mosaic/internal/adapter/secondary/hexxlastore"
	"github.com/hexxla/mosaic/internal/config"
)

func TestRunRefusesMissingDatabase(t *testing.T) {
	t.Setenv(config.EnvDBEncryptionKeyHex, "")
	t.Setenv(config.EnvDBPassphrase, "")
	t.Setenv(config.EnvPolicyFile, "")
	t.Setenv(config.EnvRatchetConfigFile, "")
	path := filepath.Join(t.TempDir(), "missing.hexxla")
	oldArgs, oldFlags := os.Args, flag.CommandLine
	t.Cleanup(func() { os.Args, flag.CommandLine = oldArgs, oldFlags })
	os.Args = []string{"mosaic-mcp", "-db", path}
	flag.CommandLine = flag.NewFlagSet("mosaic-mcp", flag.ContinueOnError)

	err := run(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("startup = %v, want os.ErrNotExist", err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 0 {
		t.Fatalf("startup created files: %v, %v", entries, err)
	}
}

func TestCloseServerDatabasePreservesShutdownErrors(t *testing.T) {
	closeErr := errors.New("close failed")
	serveErr := errors.New("serve failed")
	for _, tc := range []struct {
		name         string
		serve, close error
	}{
		{name: "close failure", close: closeErr},
		{name: "combined failures", serve: serveErr, close: closeErr},
		{name: "successful close", serve: serveErr},
		{name: "clean shutdown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := func() (err error) {
				defer closeServerDatabase(&err, func() error { return tc.close })
				return tc.serve
			}
			err := run()
			if tc.close != nil && (!errors.Is(err, tc.close) || !strings.Contains(err.Error(), "close database")) {
				t.Fatalf("run error = %v, want contextual database close error", err)
			}
			if tc.serve != nil && !errors.Is(err, tc.serve) {
				t.Fatalf("run error = %v, want server error preserved", err)
			}
			if tc.close == nil && tc.serve == nil && err != nil {
				t.Fatalf("clean shutdown returned %v", err)
			}
		})
	}
}

func TestLoadRatchetGateFromSuppliedConfig(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	gate, err := loadRatchetGate(true, "", true, log, nil)
	if err != nil {
		t.Fatalf("load supplied Ratchet config: %v", err)
	}
	if gate == nil {
		t.Fatal("supplied Ratchet config produced a nil gate")
	}
}

func TestLoadRatchetGateDisabled(t *testing.T) {
	gate, err := loadRatchetGate(false, "", false, slog.Default(), nil)
	if err != nil {
		t.Fatalf("load disabled Ratchet gate: %v", err)
	}
	if gate != nil {
		t.Fatal("empty Ratchet config unexpectedly enabled the gate")
	}
}

func TestBundledRatchetProfilesAndCustomOptIn(t *testing.T) {
	for _, advanced := range []bool{false, true} {
		gate, err := loadRatchetGate(true, "", advanced, slog.Default(), nil)
		if err != nil || gate == nil {
			t.Fatalf("bundled advanced=%t: %v", advanced, err)
		}
	}
	path := filepath.Join(t.TempDir(), "ratchet.yaml")
	policy := []byte("rules:\n  - tool: mosaic_hexxla_put_cell\n    after_any: [mosaic_hexxla_list_tags]\n    consume: true\n  - tool: mosaic_hexxla_delete_cell\n    after_any: [mosaic_hexxla_get_cell]\n")
	if err := os.WriteFile(path, policy, 0o600); err != nil {
		t.Fatal(err)
	}
	if gate, err := loadRatchetGate(false, path, false, slog.Default(), nil); err != nil || gate == nil {
		t.Fatalf("custom file did not enable core gate: %v", err)
	}
	if _, err := loadRatchetGate(false, path, true, slog.Default(), nil); err == nil {
		t.Fatal("advanced mutations accepted without policy coverage")
	}
	if err := os.WriteFile(path, append(policy, []byte("  - tool: misspelled_tool\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadRatchetGate(false, path, false, slog.Default(), nil); err == nil || !strings.Contains(err.Error(), "misspelled_tool") {
		t.Fatalf("unknown inactive policy tool accepted: %v", err)
	}
}

func TestLexicalOnlyDoesNotAdoptUnknownNativeVectors(t *testing.T) {
	db, err := hexxladb.Open(filepath.Join(t.TempDir(), "native.hexxla"), &hexxladb.Options{EmbeddingDimension: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Update(func(tx *hexxladb.Tx) error {
		key, _ := hexxladb.Pack(hexxladb.Coord{})
		return tx.PutEmbedding(key, []float32{1, 0})
	}); err != nil {
		t.Fatal(err)
	}
	if err := ensureServerEmbeddingModel(db, "model", false); !errors.Is(err, hexxlastore.ErrEmbeddingModelMismatch) {
		t.Fatalf("unknown model accepted=%v", err)
	}
	if err := ensureServerEmbeddingModel(db, "model", true); err != nil {
		t.Fatal(err)
	}
	if err := ensureServerEmbeddingModel(db, "model", false); !errors.Is(err, hexxlastore.ErrEmbeddingModelMismatch) {
		t.Fatal("lexical-only silently adopted native vector model")
	}
}
