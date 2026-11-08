package hexxlastore

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/hexxla/hexxladb"
)

// LiveDB owns an open [*hexxladb.DB] and keeps it open until active adapter calls finish.
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
	if err := fn(l.inner); err != nil {
		return fmt.Errorf("hexxlastore LiveDB WithRead: %w", err)
	}
	return nil
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
