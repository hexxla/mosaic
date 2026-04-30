package hexxlastore

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/config"
	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
)

// CellWriterAdapter implements [secondary.CellWriter] using (*hexxladb.DB).Update,
// [hexxladb.Pack], and HexxlaDB template helpers for stored cell records.
type CellWriterAdapter struct {
	live        *LiveDB
	primaryPath string
	reopenOpts  *hexxladb.Options

	maintain config.DeleteAutoMaintainConfig

	maintainMu    sync.Mutex
	debounceTimer *time.Timer
}

// NewCellWriterAdapter wraps a [LiveDB]. primaryPath must be the Hexxla primary file path passed to [hexxladb.Open]
// (used for post-delete compaction rename). reopenOpts should match the merged open options (encryption + MVCC retention).
func NewCellWriterAdapter(live *LiveDB, primaryPath string, reopenOpts *hexxladb.Options, maintain config.DeleteAutoMaintainConfig) *CellWriterAdapter {
	return &CellWriterAdapter{live: live, primaryPath: primaryPath, reopenOpts: reopenOpts, maintain: maintain}
}

// PutCell implements [secondary.CellWriter].
func (a *CellWriterAdapter) PutCell(ctx context.Context, cmd *domain.PutCellCommand) error {
	if a == nil || a.live == nil {
		return fmt.Errorf("hexxlastore cell writer: nil database")
	}
	if cmd == nil {
		return fmt.Errorf("hexxlastore cell writer: nil command")
	}
	pk, err := hexxladb.Pack(hexxladb.Coord{Q: cmd.Coord.Q, R: cmd.Coord.R})
	if err != nil {
		return fmt.Errorf("hexxlastore put cell: %w", err)
	}
	kind := cmd.Kind
	if kind == "" {
		kind = domain.CellPutKindFact
	}
	return a.live.WithRead(func(db *hexxladb.DB) error {
		updErr := db.Update(func(tx *hexxladb.Tx) error {
			switch kind {
			case domain.CellPutKindFact:
				r := hexxladb.NewFactCell(pk, cmd.RawContent, cmd.SourceID, "mcp-cell", cmd.Confidence)
				r.Tags = mergeTagLists(r.Tags, cmd.Tags)
				if err := tx.PutCell(ctx, r); err != nil {
					return fmt.Errorf("tx PutCell: %w", err)
				}
			case domain.CellPutKindUserMessage:
				r := hexxladb.NewUserMessageCell(pk, cmd.RawContent, cmd.SourceID, cmd.Confidence)
				r.Tags = mergeTagLists(r.Tags, cmd.Tags)
				if err := tx.PutCell(ctx, r); err != nil {
					return fmt.Errorf("tx PutCell: %w", err)
				}
			case domain.CellPutKindAssistantResponse:
				r := hexxladb.NewAssistantResponseCell(pk, cmd.RawContent, cmd.SourceID, cmd.Confidence)
				r.Tags = mergeTagLists(r.Tags, cmd.Tags)
				if err := tx.PutCell(ctx, r); err != nil {
					return fmt.Errorf("tx PutCell: %w", err)
				}
			default:
				return fmt.Errorf("unknown cell kind %q", kind)
			}
			return nil
		})
		if updErr != nil {
			return fmt.Errorf("hexxlastore put cell: %w", updErr)
		}
		return nil
	})
}

// PutEmbedding implements [secondary.CellWriter].
func (a *CellWriterAdapter) PutEmbedding(ctx context.Context, coord domain.AxialCoord, vec []float32) error {
	if a == nil || a.live == nil {
		return fmt.Errorf("hexxlastore cell writer: nil database")
	}
	pk, err := hexxladb.Pack(hexxladb.Coord{Q: coord.Q, R: coord.R})
	if err != nil {
		return fmt.Errorf("hexxlastore put embedding: %w", err)
	}
	return a.live.WithRead(func(db *hexxladb.DB) error {
		dim := db.EmbeddingDimension()
		if dim == 0 {
			return fmt.Errorf("hexxlastore put embedding: %w", hexxladb.ErrEmbeddingsDisabled)
		}
		if len(vec) != int(dim) {
			return fmt.Errorf("hexxlastore put embedding: vector length %d, want %d", len(vec), dim)
		}
		updErr := db.Update(func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context: %w", err)
			}
			if err := tx.PutEmbedding(pk, vec); err != nil {
				return fmt.Errorf("tx PutEmbedding: %w", err)
			}
			return nil
		})
		if updErr != nil {
			if errors.Is(updErr, hexxladb.ErrEmbeddingsDisabled) {
				return updErr
			}
			return fmt.Errorf("hexxlastore put embedding: %w", updErr)
		}
		return nil
	})
}

// DeleteCell implements [secondary.CellWriter].
func (a *CellWriterAdapter) DeleteCell(ctx context.Context, cmd *domain.DeleteCellCommand) (cellRemoved bool, err error) {
	if a == nil || a.live == nil {
		return false, fmt.Errorf("hexxlastore cell writer: nil database")
	}
	if cmd == nil {
		return false, fmt.Errorf("hexxlastore cell writer: nil command")
	}
	pk, err := hexxladb.Pack(hexxladb.Coord{Q: cmd.Coord.Q, R: cmd.Coord.R})
	if err != nil {
		return false, fmt.Errorf("hexxlastore delete cell: %w", err)
	}
	var removed bool
	delErr := a.live.WithRead(func(db *hexxladb.DB) error {
		return db.Update(func(tx *hexxladb.Tx) error {
			_, had, err := tx.GetCell(pk)
			if err != nil {
				return fmt.Errorf("tx GetCell: %w", err)
			}
			if !had {
				return nil
			}
			if err := tx.DeleteCell(ctx, pk); err != nil {
				return fmt.Errorf("tx DeleteCell: %w", err)
			}
			removed = true
			return nil
		})
	})
	if delErr != nil {
		return false, fmt.Errorf("hexxlastore delete cell: %w", delErr)
	}
	// Maintain after WithRead completes so prune/compact do not nest RLocks on LiveDB.mu.
	if !removed || !a.maintain.Enabled {
		return removed, nil
	}
	if a.maintain.DebounceAfterDelete <= 0 {
		if err := runPostDeleteMaintain(ctx, a.live, a.primaryPath, a.reopenOpts, a.maintain); err != nil {
			return removed, err
		}
		return removed, nil
	}
	a.scheduleDebouncedMaintain()
	return removed, nil
}

// scheduleDebouncedMaintain resets a timer so rapid deletes coalesce into one prune/compact pass.
func (a *CellWriterAdapter) scheduleDebouncedMaintain() {
	d := a.maintain.DebounceAfterDelete
	if d <= 0 {
		return
	}
	a.maintainMu.Lock()
	defer a.maintainMu.Unlock()
	if a.debounceTimer != nil {
		a.debounceTimer.Stop()
	}
	a.debounceTimer = time.AfterFunc(d, a.fireDebouncedMaintain)
}

func (a *CellWriterAdapter) fireDebouncedMaintain() {
	a.maintainMu.Lock()
	a.debounceTimer = nil
	a.maintainMu.Unlock()
	runCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := runPostDeleteMaintain(runCtx, a.live, a.primaryPath, a.reopenOpts, a.maintain); err != nil {
		slog.Default().Error("hexxlastore debounced post-delete maintain", "err", err)
	}
}

// FlushPostDeleteMaintain stops any pending debounced maintenance and runs prune/compact once before shutdown.
// Call while the database is still open (e.g. defer before [LiveDB.Close]).
func (a *CellWriterAdapter) FlushPostDeleteMaintain(ctx context.Context) error {
	if a == nil || !a.maintain.Enabled {
		return nil
	}
	a.maintainMu.Lock()
	t := a.debounceTimer
	a.debounceTimer = nil
	a.maintainMu.Unlock()
	if t == nil {
		return nil
	}
	t.Stop()
	return runPostDeleteMaintain(ctx, a.live, a.primaryPath, a.reopenOpts, a.maintain)
}

func mergeTagLists(base, extra []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(base)+len(extra))
	for _, t := range base {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	for _, t := range extra {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	slices.Sort(out)
	return out
}

// Ensure CellWriterAdapter implements secondary.CellWriter.
var _ secondary.CellWriter = (*CellWriterAdapter)(nil)
