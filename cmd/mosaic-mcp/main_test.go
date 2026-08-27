package main

import (
	"io"
	"log/slog"
	"testing"
)

func TestLoadRatchetGateFromSuppliedConfig(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	gate, err := loadRatchetGate("../../configs/ratchet.yaml", log)
	if err != nil {
		t.Fatalf("load supplied Ratchet config: %v", err)
	}
	if gate == nil {
		t.Fatal("supplied Ratchet config produced a nil gate")
	}
}

func TestLoadRatchetGateDisabled(t *testing.T) {
	gate, err := loadRatchetGate("", slog.Default())
	if err != nil {
		t.Fatalf("load disabled Ratchet gate: %v", err)
	}
	if gate != nil {
		t.Fatal("empty Ratchet config unexpectedly enabled the gate")
	}
}
