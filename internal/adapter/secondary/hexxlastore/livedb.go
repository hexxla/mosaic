package hexxlastore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/hexxla/hexxladb"
)

// LiveDB holds the current [*hexxladb.DB] with an RWMutex so post-delete compaction
// can exclusively swap the handle after rewriting the primary file on disk.
type LiveDB struct {
	mu sync.RWMutex

	closed atomic.Bool
	inner  *hexxladb.DB
}

// NewLiveDB wraps an opened database handle. Call [LiveDB.Close] at shutdown instead of [*hexxladb.DB.Close] directly.
func NewLiveDB(db *hexxladb.DB) *LiveDB {
	return &LiveDB{inner: db}
}

// WithRead runs fn with the current database pointer under a shared lock (concurrent-safe with other reads).
func (l *LiveDB) WithRead(fn func(*hexxladb.DB) error) error {
	if l == nil {
		return fmt.Errorf("hexxlastore LiveDB: nil")
	}
	if l.closed.Load() {
		return fmt.Errorf("hexxlastore LiveDB: closed")
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.closed.Load() || l.inner == nil {
		return fmt.Errorf("hexxlastore LiveDB: closed")
	}
	return fn(l.inner)
}

// Close closes the current inner database (exclusive with readers).
func (l *LiveDB) Close() error {
	if l == nil {
		return nil
	}
	if !l.closed.CompareAndSwap(false, true) {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	var err error
	if l.inner != nil {
		err = l.inner.Close()
		l.inner = nil
	}
	return err
}

// CompactSwap runs [(*hexxladb.DB).Compact] to a temp file, closes the live handle,
// replaces the primary file atomically via [os.Rename] (POSIX replace), opens a replacement
// handle with reopenOpts, and swaps [LiveDB.inner]. Holds an exclusive mutex for the duration.
//
// Callers must reproduce encryption and MVCC retention in reopenOpts on each reopen.
func (l *LiveDB) CompactSwap(ctx context.Context, primaryPath string, reopenOpts *hexxladb.Options) error {
	if l == nil {
		return fmt.Errorf("hexxlastore LiveDB: nil")
	}
	if l.closed.Load() {
		return fmt.Errorf("hexxlastore LiveDB: closed")
	}

	dir := filepath.Dir(primaryPath)
	base := filepath.Base(primaryPath)
	tmp, err := os.CreateTemp(dir, base+".compact-*")
	if err != nil {
		return fmt.Errorf("hexxlastore compact swap: tempfile: %w", err)
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("hexxlastore compact swap: close tempfile writer: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed.Load() || l.inner == nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("hexxlastore LiveDB: closed")
	}
	old := l.inner

	if err := old.Compact(ctx, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("hexxlastore compact swap: compact: %w", err)
	}
	if err := old.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("hexxlastore compact swap: close old: %w", err)
	}

	if err := os.Rename(tmpPath, primaryPath); err != nil {
		l.inner = nil
		return fmt.Errorf("hexxlastore compact swap: publish compacted primary: %w", err)
	}

	// The compacted file is authoritative; WAL from the superseded generation would mis-apply to the new tree.
	walLegacy := primaryPath + "-wal"
	_ = os.Remove(walLegacy)
	_ = os.Remove(tmpPath + "-wal")

	newDB, err := hexxladb.Open(primaryPath, reopenOpts)
	if err != nil {
		l.inner = nil
		return fmt.Errorf("hexxlastore compact swap: reopen: %w", err)
	}
	l.inner = newDB
	return nil
}
