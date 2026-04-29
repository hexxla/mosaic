package config

import (
	"fmt"
	"os"
	"strings"
)

// EnvDBPath is the environment variable for the HexxlaDB file path (used by cmd/mosaic-mcp and path resolution).
const EnvDBPath = "MOSAIC_DB_PATH"

// DB holds paths required to open the embedded Hexxla database for MCP tools.
type DB struct {
	// Path is the filesystem path to the HexxlaDB database file (create if missing).
	Path string
}

// LoadDBFromEnv reads MOSAIC_DB_PATH. The value must be a non-empty path string.
func LoadDBFromEnv() (DB, error) {
	p := strings.TrimSpace(os.Getenv(EnvDBPath))
	if p == "" {
		return DB{}, fmt.Errorf("%s must be set to a HexxlaDB file path", EnvDBPath)
	}
	return DB{Path: p}, nil
}
