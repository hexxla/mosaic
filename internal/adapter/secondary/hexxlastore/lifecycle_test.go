package hexxlastore

import (
	"errors"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/config"
)

func openLifecycleDB(t *testing.T) (*hexxladb.DB, *LiveDB) {
	t.Helper()
	db, err := hexxladb.Open(filepath.Join(t.TempDir(), "lifecycle.hexxla"), &hexxladb.Options{
		EmbeddingDimension: 2,
		MVCCRetention:      hexxladb.MVCCRetention{RetainCommitsBehindHead: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	live := NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })
	return db, live
}

func TestLiveDBCloseWaitsForActiveCallAndRejectsNewCalls(t *testing.T) {
	t.Parallel()
	_, live := openLifecycleDB(t)
	entered, release := make(chan struct{}), make(chan struct{})
	releaseRead := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseRead)
	readDone := make(chan error, 1)
	go func() {
		readDone <- live.WithRead(func(*hexxladb.DB) error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	closeDone := make(chan error, 1)
	go func() { closeDone <- live.Close() }()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for !live.closed.Load() {
		select {
		case <-deadline.C:
			t.Fatal("Close did not begin")
		default:
			runtime.Gosched()
		}
	}
	if err := live.WithRead(func(*hexxladb.DB) error {
		t.Error("new work entered after Close began")
		return nil
	}); err == nil {
		t.Fatal("new work was accepted during Close")
	}
	select {
	case err := <-closeDone:
		t.Fatalf("Close finished while callback was active: %v", err)
	default:
	}
	releaseRead()
	if err := <-readDone; err != nil {
		t.Fatal(err)
	}
	if err := <-closeDone; err != nil {
		t.Fatal(err)
	}
	if err := live.Close(); err != nil {
		t.Fatalf("repeated Close: %v", err)
	}
}

func TestFlushRunsPendingMaintenanceAndStopsFutureScheduling(t *testing.T) {
	t.Parallel()
	_, live := openLifecycleDB(t)
	writer := NewCellWriterAdapter(live, config.DeleteAutoMaintainConfig{
		Enabled: true, Prune: true, MVCCPruneProfile: "invalid-profile", DebounceAfterDelete: time.Hour,
	})
	writer.scheduleMaintenance(t.Context())
	t.Cleanup(func() { _ = writer.FlushPostDeleteMaintain(t.Context()) })
	if err := writer.FlushPostDeleteMaintain(t.Context()); !errors.Is(err, hexxladb.ErrInvalidArgument) {
		t.Fatalf("pending maintenance was not flushed: %v", err)
	}
	writer.scheduleMaintenance(t.Context())
	if err := writer.FlushPostDeleteMaintain(t.Context()); err != nil {
		t.Fatalf("maintenance was scheduled again after shutdown: %v", err)
	}
	if writer.debounceTimer != nil {
		t.Fatal("shutdown left a timer scheduled")
	}
}
