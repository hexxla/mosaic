//go:build linux

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hexxla/hexxladb"
)

func TestQualificationStagesAreExplicit(t *testing.T) {
	c := options{Dir: "unused", Owner: "void", Mode: "lexical", Duration: 30 * time.Minute, Interval: time.Second, Maintenance: 10 * time.Minute}
	for _, d := range []time.Duration{30 * time.Minute, time.Hour, 24 * time.Hour} {
		c.Duration = d
		if err := validate(c); err != nil {
			t.Fatal(err)
		}
	}
	c.Duration = 2 * time.Hour
	if validate(c) == nil {
		t.Fatal("unqualified stage accepted")
	}
	c.Duration = 30 * time.Minute
	c.Mode = "ollama"
	c.URL = "http://127.0.0.1:11435"
	c.Dim = 384
	if validate(c) == nil {
		t.Fatal("live provider accepted without digest")
	}
}
func TestResourceLimitsFailClosed(t *testing.T) {
	group := t.TempDir()
	for name, value := range map[string]string{"memory.max": "max", "memory.swap.max": "0", "cpu.max": "100000 100000"} {
		if err := os.WriteFile(filepath.Join(group, name), []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if checkGroup(group, gib) == nil {
		t.Fatal("unbounded memory accepted")
	}
	if err := os.WriteFile(filepath.Join(group, "memory.max"), []byte("1073741824"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkGroup(group, gib); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(group, "cpu.max"), []byte("200000 100000"), 0o600); err != nil {
		t.Fatal(err)
	}
	if checkGroup(group, gib) == nil {
		t.Fatal("excessive CPU quota accepted")
	}
}
func TestCompactionRetainsOracleAndConsumer(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "a"), 0o700); err != nil {
		t.Fatal(err)
	}
	w := &workload{cfg: options{Dir: dir, Mode: "lexical"}, expected: map[int]string{0: "qualification baseline"}, opts: &hexxladb.Options{ChangelogEnabled: true}, path: filepath.Join(dir, "a", "live.db")}
	var err error
	w.db, err = hexxladb.Open(w.path, w.opts)
	if err != nil {
		t.Fatal(err)
	}
	key, err := hexxladb.Pack(hexxladb.Coord{})
	if err != nil {
		t.Fatal(err)
	}
	if err = w.db.Update(func(tx *hexxladb.Tx) error {
		return tx.PutCell(ctx, hexxladb.NewFactCell(key, w.expected[0], "qualification", "baseline", 1))
	}); err != nil {
		t.Fatal(err)
	}
	w.connect(ctx)
	t.Cleanup(func() {
		if w.wire != nil {
			w.wire.close()
		}
		if w.live != nil {
			if err := w.live.Close(); err != nil {
				t.Error(err)
			}
		}
	})
	for range 2 {
		w.maintain(ctx)
	}
	if w.out.Compactions != 2 || w.out.Backups != 2 || w.out.Denials != 3 {
		t.Fatalf("maintenance not exercised: %+v", w.out)
	}
	if _, ok, err := w.db.GetChangelogConsumerCursor("soak"); err != nil || !ok {
		t.Fatalf("consumer lost: %v", err)
	}
	// The content oracle must reject a readable, internally healthy wrong value.
	if err = w.db.Update(func(tx *hexxladb.Tx) error {
		return tx.PutCell(ctx, hexxladb.NewFactCell(key, "wrong", "qualification", "baseline", 1))
	}); err != nil {
		t.Fatal(err)
	}
	rejected := func() (rejected bool) {
		defer func() { rejected = recover() != nil }()
		w.oracle(ctx, w.db)
		return false
	}()
	if !rejected {
		t.Fatal("incorrect data passed qualification oracle")
	}
}

func TestLiveProviderRequiresPinnedCPUModel(t *testing.T) {
	var gpu atomic.Bool
	gpu.Store(true)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/version":
			_, _ = w.Write([]byte(`{"version":"fixture"}`))
		case "/api/tags":
			_, _ = w.Write([]byte(`{"models":[{"name":"fixture:latest","digest":"` + strings.Repeat("a", 64) + `"}]}`))
		case "/api/embeddings":
			_, _ = w.Write([]byte(`{"embedding":[1,2]}`))
		case "/api/ps":
			if gpu.Load() {
				_, _ = w.Write([]byte(`{"models":[{"name":"fixture:latest","size_vram":1}]}`))
			} else {
				_, _ = w.Write([]byte(`{"models":[{"name":"fixture:latest","size_vram":0}]}`))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	c := options{Mode: "ollama", URL: server.URL, Model: "fixture:latest", Digest: strings.Repeat("a", 64), Dim: 2}
	if _, _, err := provider(t.Context(), c); err == nil {
		t.Fatal("unbounded GPU profile accepted")
	}
	gpu.Store(false)
	if embedder, version, err := provider(t.Context(), c); err != nil || embedder == nil || version != "fixture" {
		t.Fatalf("CPU fixture rejected: %s %v", version, err)
	}
	c.Digest = strings.Repeat("b", 64)
	if _, _, err := provider(t.Context(), c); err == nil {
		t.Fatal("retagged model accepted")
	}
}

func TestSummarySerializesStageDurations(t *testing.T) {
	if err := writeJSON(filepath.Join(t.TempDir(), "summary.json"), summary{Options: options{Duration: 30 * time.Minute, Interval: time.Second, Maintenance: 10 * time.Minute}}); err != nil {
		t.Fatal(err)
	}
}

func TestLauncherMatchesOSDeadlineToStage(t *testing.T) {
	fake := t.TempDir()
	//nolint:gosec // G306: this private test fixture must be executable to intercept the launcher.
	if err := os.WriteFile(filepath.Join(fake, "systemd-run"), []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fake+":"+os.Getenv("PATH"))
	for _, tc := range []struct {
		args     []string
		deadline string
	}{
		{[]string{"--duration=1h", "--preflight"}, "4200"},
		{[]string{"--duration", "24h", "--preflight"}, "87000"},
		{[]string{"--duration=24h", "--duration", "30m", "--preflight"}, "2400"},
		{[]string{"--verify", "--duration=5s"}, "720"},
	} {
		//nolint:gosec // G204: arguments are fixed test cases; the systemd executable is replaced by the private fixture.
		out, err := exec.CommandContext(t.Context(), "bash", append([]string{"../../scripts/soak.sh"}, tc.args...)...).CombinedOutput()
		if err != nil || !strings.Contains(string(out), "RuntimeMaxSec="+tc.deadline) {
			t.Fatalf("args=%v deadline=%s output=%s err=%v", tc.args, tc.deadline, out, err)
		}
	}
	if err := exec.CommandContext(t.Context(), "bash", "../../scripts/soak.sh", "--duration=2h").Run(); err == nil {
		t.Fatal("unknown stage reached launcher")
	}
}
