package config_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sploitzberg/mosaic/internal/config"
)

func TestMosaicConfigRejectsUnknownFieldsAndTrailingDocuments(t *testing.T) {
	t.Parallel()
	for name, raw := range map[string]string{
		"retention enforcement typo": "version: 1\nretention:\n  capture_mode: none\n  enforcment: true\n",
		"database option typo":       "version: 1\ndatabase:\n  passpharse: example\n",
		"retrieval cap typo":         "version: 1\nretrieval:\n  session_approx_token_budgt: 10\n",
		"root typo":                  "version: 1\nretenton:\n  capture_mode: none\n",
		"second document":            "version: 1\n---\nretention:\n  capture_mode: none\n",
		"empty second document":      "version: 1\n---\n",
		"malformed second document":  "version: 1\n---\n[\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := config.ParseMosaicConfigYAML([]byte(raw)); err == nil {
				t.Fatal("invalid configuration was accepted")
			}
		})
	}
}

func TestMosaicConfigStrictDecodingRejectsPolicyAliases(t *testing.T) {
	t.Parallel()
	for _, policy := range []string{"persistence_policy: {capture_mode: none}", "capture_mode: none", "enforcement: reject", "notes: legacy", "retention: {enforcement: reject}", "retention: {enforcement: off}", "retention: {enforcement: 'true'}"} {
		if _, err := config.ParseMosaicConfigYAML([]byte("version: 1\n" + policy)); err == nil {
			t.Fatalf("legacy policy accepted: %s", policy)
		}
	}
}

func TestRatchetExplicitCurrentDirectoryIsRejected(t *testing.T) {
	t.Parallel()
	for _, path := range []string{".", "./", "a/.."} {
		if _, err := config.LoadRatchetPolicy(path); err == nil {
			t.Fatalf("explicit path %q disabled Ratchet", path)
		}
	}
	if _, err := config.LoadRatchetPolicy(""); err != nil {
		t.Fatal(err)
	}
}

func TestMCPRejectsPatternAndNoncanonicalPaths(t *testing.T) {
	t.Setenv("MOSAIC_MCP_ADDR", "127.0.0.1:8787")
	for _, path := range []string{"/", "/mcp/{", "/mcp/{id}", "/mcp/{rest...}", "/mcp/{$}", "/a/../mcp", "//mcp", "/mcp/", "/mcp?x=1", "/mcp#fragment", "/mcp space", "/mcp\npath", "/m%63p", "/mcp%"} {
		t.Run(path, func(t *testing.T) {
			t.Setenv("MOSAIC_MCP_PATH", path)
			if _, err := config.LoadMCPFromEnv(); err == nil {
				t.Fatalf("invalid endpoint path %q was accepted", path)
			}
		})
	}
	for _, path := range []string{"/mcp", "/api/mcp-v1"} {
		t.Setenv("MOSAIC_MCP_PATH", path)
		got, err := config.LoadMCPFromEnv()
		if err != nil {
			t.Fatal(err)
		}
		mux := http.NewServeMux()
		mux.HandleFunc("POST "+got.Path, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, http.NoBody))
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("literal path %q response=%d", path, recorder.Code)
		}
	}
}

func TestResolveOllamaRejectsInvalidBasesWithoutDisclosingSecrets(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		"ftp://operator:secret@example.test",
		"http://operator:secret@example.test?token=secret",
		"http://operator:secret@example.test?",
		"http://operator:secret@example.test#secret",
		"http://operator:secret@example.test#",
		"http://operator:secret@example.test:bad",
		"http://operator:secret@/prefix",
	} {
		_, err := config.ResolveOllama(config.OllamaResolveInput{FlagBaseURL: raw})
		if err == nil {
			t.Fatalf("invalid base %q was accepted", raw)
		}
		if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "operator") {
			t.Fatalf("configuration error disclosed credentials: %v", err)
		}
	}
	_, err := config.ResolveOllama(config.OllamaResolveInput{FlagBaseURL: "http://operator:secret@example.test:bad"})
	if _, ok := errors.AsType[*url.Error](err); !ok {
		t.Fatalf("URL parse cause is no longer inspectable: %v", err)
	}
}

func TestResolveOllamaPreservesHTTPBasePrefixesAndCredentials(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{"http://127.0.0.1:11434", "https://operator:secret@example.test/ollama/"} {
		got, err := config.ResolveOllama(config.OllamaResolveInput{FlagBaseURL: raw})
		if err != nil || got.Base.String() != raw {
			t.Fatalf("supported base %q: %+v, %v", raw, got, err)
		}
	}
}
