package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadRatchetObservabilityDisabled(t *testing.T) {
	t.Parallel()

	got, err := LoadRatchetObservability("", "")
	if err != nil {
		t.Fatalf("LoadRatchetObservability: %v", err)
	}
	if got.Enabled {
		t.Fatal("empty token path must disable observability")
	}
}

func TestLoadRatchetObservability(t *testing.T) {
	t.Parallel()

	tokenFile := filepath.Join(t.TempDir(), "observer.token")
	wantToken := strings.Repeat("a", minObservabilityTokenBytes)
	if err := os.WriteFile(tokenFile, []byte(wantToken+"\n"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}

	got, err := LoadRatchetObservability(tokenFile, "")
	if err != nil {
		t.Fatalf("LoadRatchetObservability: %v", err)
	}
	if !got.Enabled || got.Path != DefaultRatchetObservabilityPath || string(got.BearerToken) != wantToken {
		t.Fatalf("unexpected config: %+v", got)
	}
}

func TestLoadRatchetObservabilityRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		path  string
		mode  os.FileMode
		token string
	}{
		{name: "relative path", path: "stream", mode: 0o600, token: strings.Repeat("a", minObservabilityTokenBytes)},
		{name: "mux pattern", path: "/stream/{session}", mode: 0o600, token: strings.Repeat("a", minObservabilityTokenBytes)},
		{name: "short token", path: "/stream", mode: 0o600, token: "short"},
	}
	if runtime.GOOS != "windows" {
		tests = append(tests, struct {
			name  string
			path  string
			mode  os.FileMode
			token string
		}{name: "public token file", path: "/stream", mode: 0o644, token: strings.Repeat("a", minObservabilityTokenBytes)})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tokenFile := filepath.Join(t.TempDir(), "observer.token")
			if err := os.WriteFile(tokenFile, []byte(tt.token), tt.mode); err != nil {
				t.Fatalf("write token: %v", err)
			}
			if _, err := LoadRatchetObservability(tokenFile, tt.path); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
