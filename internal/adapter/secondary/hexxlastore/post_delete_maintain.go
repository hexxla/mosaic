package hexxlastore

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/config"
)

// runPostDeleteMaintain runs bounded MVCC pruning after a committed delete.
func runPostDeleteMaintain(ctx context.Context, live *LiveDB, cfg config.DeleteAutoMaintainConfig) error {
	if live == nil || !cfg.Enabled {
		return nil
	}
	prof := cfg.MVCCPruneProfile
	if prof == "" {
		prof = hexxladb.MVCCPruneBalanced
	}
	if cfg.Prune {
		err := live.WithRead(func(db *hexxladb.DB) error {
			sched := hexxladb.PruneScheduler{Profile: prof}
			rounds := cfg.PruneRoundsCap()
			for range rounds {
				if err := ctx.Err(); err != nil {
					return fmt.Errorf("post-delete prune: %w", err)
				}
				n, err := sched.Tick(db)
				if err != nil {
					return fmt.Errorf("prune tick: %w", err)
				}
				if n == 0 {
					return nil
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("hexxlastore post-delete prune: %w", err)
		}
	}
	return nil
}

// scheduleMaintenance coalesces deletes. The mutex also serializes shutdown
// with a running callback and prevents a superseded timer from doing work.
func (a *CellWriterAdapter) scheduleMaintenance(ctx context.Context) {
	a.maintainMu.Lock()
	defer a.maintainMu.Unlock()
	if a.maintenanceStopped {
		return
	}
	if a.maintain.DebounceAfterDelete <= 0 {
		a.reportMaintenance(ctx)
		return
	}
	if a.debounceTimer != nil {
		a.debounceTimer.Stop()
	}
	a.maintenanceGeneration++
	generation := a.maintenanceGeneration
	a.debounceTimer = time.AfterFunc(a.maintain.DebounceAfterDelete, func() {
		a.maintainMu.Lock()
		defer a.maintainMu.Unlock()
		if a.maintenanceStopped || a.maintenanceGeneration != generation {
			return
		}
		a.debounceTimer = nil
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		a.reportMaintenance(ctx)
	})
}

func (a *CellWriterAdapter) reportMaintenance(ctx context.Context) {
	if err := runPostDeleteMaintain(ctx, a.live, a.maintain); err != nil {
		slog.ErrorContext(ctx, "post-delete maintenance failed after delete committed", "err", err)
	}
}

// FlushPostDeleteMaintain waits for active pruning and runs any pending pass
// before the database is closed.
func (a *CellWriterAdapter) FlushPostDeleteMaintain(ctx context.Context) error {
	a.maintainMu.Lock()
	defer a.maintainMu.Unlock()
	a.maintenanceStopped = true
	if a.debounceTimer == nil {
		return nil
	}
	a.debounceTimer.Stop()
	a.debounceTimer = nil
	return runPostDeleteMaintain(ctx, a.live, a.maintain)
}
