package hexxlastore

import (
	"context"
	"fmt"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/config"
)

// runPostDeleteMaintain runs bounded MVCC prune passes and/or exclusive compact after a cell was removed.
func runPostDeleteMaintain(ctx context.Context, live *LiveDB, primaryPath string, reopenOpts *hexxladb.Options, cfg config.DeleteAutoMaintainConfig) error {
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
			for i := 0; i < rounds; i++ {
				if err := ctx.Err(); err != nil {
					return err
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
	if cfg.Compact {
		if err := live.CompactSwap(ctx, primaryPath, reopenOpts); err != nil {
			return fmt.Errorf("hexxlastore post-delete compact: %w", err)
		}
	}
	return nil
}
