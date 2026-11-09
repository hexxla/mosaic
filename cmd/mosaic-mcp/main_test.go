package main

import (
	"errors"
	"flag"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/sploitzberg/mosaic/internal/config"
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

func TestLoadRatchetGateFromSuppliedConfig(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	gate, err := loadRatchetGate("../../configs/ratchet.yaml", log, nil)
	if err != nil {
		t.Fatalf("load supplied Ratchet config: %v", err)
	}
	if gate == nil {
		t.Fatal("supplied Ratchet config produced a nil gate")
	}
}

func TestLoadRatchetGateDisabled(t *testing.T) {
	gate, err := loadRatchetGate("", slog.Default(), nil)
	if err != nil {
		t.Fatalf("load disabled Ratchet gate: %v", err)
	}
	if gate != nil {
		t.Fatal("empty Ratchet config unexpectedly enabled the gate")
	}
}
