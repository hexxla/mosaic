package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// EnvMosaicDBDir is the directory used when resolving -name to a file (<dir>/<name>.hexxla).
// Default when unset: ".tmp" (see [DefaultMosaicDBDir]).
const EnvMosaicDBDir = "MOSAIC_DB_DIR" //nolint:gosec // G101: environment variable name, not a secret value

// MosaicDefaultRelDBFile is the default HexxlaDB path when no -db, -name, or MOSAIC_DB_PATH is set
// (create-db / mosaic-seed only). It is resolved relative to the process working directory (shell cwd).
const MosaicDefaultRelDBFile = "mosaic.hexxla"

var mosaicDBBaseNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

// MosaicDBPathInput holds -db / -name / -db-dir flags for [ResolveMosaicDBPath].
type MosaicDBPathInput struct {
	DBFlag    string
	NameFlag  string
	DBDirFlag string
}

// DefaultMosaicDBDir returns MOSAIC_DB_DIR when set, else ".tmp".
func DefaultMosaicDBDir() string {
	d := strings.TrimSpace(os.Getenv(EnvMosaicDBDir))
	if d != "" {
		return filepath.Clean(d)
	}
	return ".tmp"
}

// ValidateMosaicDBBaseName ensures -name is a single path segment (no separators or "..").
func ValidateMosaicDBBaseName(name string) error {
	s := strings.TrimSpace(name)
	if s == "" {
		return errors.New("config: -name must be non-empty")
	}
	if strings.Contains(s, string(filepath.Separator)) || strings.Contains(s, "/") || strings.Contains(s, "\\") {
		return fmt.Errorf("config: -name must not contain path separators (use -db for a full path): %q", s)
	}
	if strings.Contains(s, "..") {
		return fmt.Errorf("config: invalid -name %q", s)
	}
	if !mosaicDBBaseNameRe.MatchString(s) {
		return fmt.Errorf("config: -name must match [a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}: %q", s)
	}
	return nil
}

// ResolveMosaicDBPath chooses the database file path.
//
// Precedence: -db > -name (under -db-dir / MOSAIC_DB_DIR) > MOSAIC_DB_PATH > optional default file.
//
// When defaultWhenUnset is true (mosaic-create-db, mosaic-seed), a missing path falls back to [MosaicDefaultRelDBFile].
// When false (mosaic-mcp), MOSAIC_DB_PATH or -db or -name is required.
func ResolveMosaicDBPath(in MosaicDBPathInput, defaultWhenUnset bool) (string, error) {
	db := strings.TrimSpace(in.DBFlag)
	name := strings.TrimSpace(in.NameFlag)
	if db != "" && name != "" {
		return "", errors.New("config: use either -db or -name, not both")
	}
	if db != "" {
		return filepath.Clean(db), nil
	}
	if name != "" {
		if err := ValidateMosaicDBBaseName(name); err != nil {
			return "", err
		}
		dir := strings.TrimSpace(in.DBDirFlag)
		if dir == "" {
			dir = DefaultMosaicDBDir()
		} else {
			dir = filepath.Clean(dir)
		}
		return filepath.Join(dir, name+".hexxla"), nil
	}
	if p := strings.TrimSpace(os.Getenv(EnvDBPath)); p != "" {
		return filepath.Clean(p), nil
	}
	if defaultWhenUnset {
		return filepath.Clean(MosaicDefaultRelDBFile), nil
	}
	return "", fmt.Errorf("config: set %s or pass -db or -name", EnvDBPath)
}
