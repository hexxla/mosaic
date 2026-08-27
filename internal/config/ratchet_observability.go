package config

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
)

const (
	// DefaultRatchetObservabilityPath is the opt-in live event stream path.
	DefaultRatchetObservabilityPath = "/observability/stream"
	maxObservabilityTokenBytes      = 4096
	minObservabilityTokenBytes      = 32
)

// RatchetObservability holds the validated configuration for the live Ratchet
// event stream. An empty token-file path disables the stream.
type RatchetObservability struct {
	Enabled     bool
	Path        string
	BearerToken []byte
}

// LoadRatchetObservability validates the stream path and reads a private bearer
// token file. The token is never accepted through a command-line value or URL.
func LoadRatchetObservability(tokenFile, streamPath string) (RatchetObservability, error) {
	tokenFile = strings.TrimSpace(tokenFile)
	if tokenFile == "" {
		return RatchetObservability{}, nil
	}

	streamPath = strings.TrimSpace(streamPath)
	if streamPath == "" {
		streamPath = DefaultRatchetObservabilityPath
	}
	if !strings.HasPrefix(streamPath, "/") || path.Clean(streamPath) != streamPath || strings.ContainsAny(streamPath, "{}?#") {
		return RatchetObservability{}, fmt.Errorf("invalid Ratchet observability path %q", streamPath)
	}

	token, err := loadPrivateTokenFile(tokenFile)
	if err != nil {
		return RatchetObservability{}, err
	}
	return RatchetObservability{Enabled: true, Path: streamPath, BearerToken: token}, nil
}

func loadPrivateTokenFile(tokenFile string) ([]byte, error) {
	info, err := os.Lstat(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("stat Ratchet observability token file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("ratchet observability token file %q must be a regular file", tokenFile)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("ratchet observability token file %q permissions must not grant group or other access", tokenFile)
	}
	if info.Size() > maxObservabilityTokenBytes {
		return nil, fmt.Errorf("ratchet observability token file %q exceeds %d bytes", tokenFile, maxObservabilityTokenBytes)
	}

	// #nosec G304 -- the operator explicitly selects this credential path.
	raw, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("read Ratchet observability token file: %w", err)
	}
	token := []byte(strings.TrimSpace(string(raw)))
	if len(token) < minObservabilityTokenBytes {
		return nil, fmt.Errorf("ratchet observability token must be at least %d bytes", minObservabilityTokenBytes)
	}
	return token, nil
}
