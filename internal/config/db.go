package config

// EnvDBPath is the environment variable for the HexxlaDB file path (used by cmd/mosaic-mcp and path resolution).
const EnvDBPath = "MOSAIC_DB_PATH"

// DB holds paths required to open the embedded Hexxla database for MCP tools.
type DB struct {
	// Path is the filesystem path to the HexxlaDB database file (create if missing).
	Path string
}
