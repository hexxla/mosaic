package config

import (
	"fmt"
	"os"
	"strings"
)

const envDBPath = "MOSAIC_DB_PATH"

// DB holds paths required to open the embedded Hexxla database for MCP tools.
type DB struct {
	// Path is the filesystem path to the HexxlaDB database file (create if missing).
	Path string
}

// LoadDBFromEnv reads MOSAIC_DB_PATH. The value must be a non-empty path string.
func LoadDBFromEnv() (DB, error) {
	p := strings.TrimSpace(os.Getenv(envDBPath))
	if p == "" {
		return DB{}, fmt.Errorf("%s must be set to a HexxlaDB file path", envDBPath)
	}
	return DB{Path: p}, nil
}
