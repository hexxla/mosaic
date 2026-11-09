package hexxlastore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hexxla/hexxladb"
)

// CreateCandidate builds and checks a private database before copying its
// recovery set to an unused, offline destination.
func CreateCandidate(ctx context.Context, path string, opts *hexxladb.Options, populate func(*hexxladb.DB) error) (retErr error) {
	for _, name := range []string{path, path + "-wal", path + "-changelog", path + ".rotate.state"} {
		if _, err := os.Lstat(name); err == nil {
			return fmt.Errorf("database candidate destination %q: %w", name, os.ErrExist)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect candidate destination %q: %w", name, err)
		}
	}

	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return fmt.Errorf("create candidate parent directory: %w", err)
	}
	temp, err := os.MkdirTemp(parent, ".mosaic-candidate-")
	if err != nil {
		return fmt.Errorf("create private candidate directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(temp); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("remove private candidate directory: %w", err))
		}
	}()
	db, err := hexxladb.Open(filepath.Join(temp, "candidate.hexxla"), opts)
	if err != nil {
		return fmt.Errorf("open private candidate: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("close private candidate: %w", err))
		}
	}()

	if populate != nil {
		if err := populate(db); err != nil {
			return err
		}
	}
	report, err := db.HealthCheck(ctx, hexxladb.DefaultHealthCheckConfig())
	if err != nil {
		return fmt.Errorf("validate candidate: %w", err)
	}
	if report.TagIndexErrors != 0 || report.SourceIndexErrors != 0 || len(report.OrphanedSeams) != 0 {
		return fmt.Errorf("candidate failed integrity checks: %d tag errors, %d source errors, %d orphan seams",
			report.TagIndexErrors, report.SourceIndexErrors, len(report.OrphanedSeams))
	}
	if err := db.BackupTo(ctx, path); err != nil {
		return fmt.Errorf("copy validated candidate: %w", err)
	}
	return nil
}
