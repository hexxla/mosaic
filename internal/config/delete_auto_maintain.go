package config

import (
	"fmt"
	"time"

	"github.com/hexxla/hexxladb"
)

// DefaultAutoMaintainRetainCommits is merged into Mosaic → Hexxla [hexxladb.Options.MVCCRetention]
// when post-delete pruning is enabled and the operator omitted mvcc_retain_commits_behind_head.
// It bounds how far [DB.ViewAt] can see into MVCC history after pruning via [DB.SuggestedPruneBeforeSeq].
//
// Uses 8 — shallow MVCC slack so prune+compact keeps versioned-row volume close to logical cell churn.
// Hexxla's suggested watermark is inactive while CommitSeq ≤ RetainCommitsBehindHead (beforeSeq stays 0);
// defaults like 512 made post-delete prune a no-op for typical MCP sessions.
// Raise in YAML when [DB.ViewAt] must reach farther into commit history (trades disk for time travel depth).
const DefaultAutoMaintainRetainCommits uint64 = 8

// DefaultDebounceAfterDelete is applied when auto_maintain_after_cell_delete.enabled is true
// and debounce_after_delete_ms is omitted. Rapid deletes coalesce into one prune+compact pass.
const DefaultDebounceAfterDelete = 2 * time.Second

// DeleteAutoMaintainConfig controls optional MVCC prune + compact after a successful cell delete (Mosaic hexxlastore).
type DeleteAutoMaintainConfig struct {
	Enabled              bool
	Prune                bool
	Compact              bool
	MaxPruneRoundsPerDel int // cap PruneScheduler.Tick batches per delete; default 64
	MVCCPruneProfile     hexxladb.MVCCPruneProfile
	// DebounceAfterDelete delays prune+compact until this duration elapses without another delete (0 = run immediately after each delete).
	DebounceAfterDelete time.Duration
}

// PruneRoundsCap returns MaxPruneRoundsPerDel when >0, otherwise 64.
func (c DeleteAutoMaintainConfig) PruneRoundsCap() int {
	if c.MaxPruneRoundsPerDel <= 0 {
		return 64
	}
	return c.MaxPruneRoundsPerDel
}

// ParseMVCCPruneProfile maps YAML prune_profile strings to [hexxladb.MVCCPruneProfile].
func ParseMVCCPruneProfile(s string) (hexxladb.MVCCPruneProfile, error) {
	switch s {
	case "", "balanced":
		return hexxladb.MVCCPruneBalanced, nil
	case "low-latency", "low_latency":
		return hexxladb.MVCCPruneLowLatency, nil
	case "long-history", "long_history":
		return hexxladb.MVCCPruneLongHistory, nil
	default:
		return "", fmt.Errorf("config: unknown database.auto_maintain_after_cell_delete.prune_profile %q (want balanced, low-latency, long-history)", s)
	}
}

func applyDeleteAutoMaintainDefaults(l *MosaicConfigLoaded) {
	if l == nil || !l.DeleteAutoMaintain.Enabled {
		return
	}
	d := &l.DeleteAutoMaintain
	if d.Prune && l.MVCCRetainCommitsBehindHead == 0 {
		l.MVCCRetainCommitsBehindHead = DefaultAutoMaintainRetainCommits
	}
}
