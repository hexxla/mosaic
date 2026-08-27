package hexxlastore

import (
	"context"
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
func (a *CellWriterAdapter) PutCell(ctx context.Context, cmd *domain.PutCellCommand) (domain.PutCellMutationResult, error) {
	if a == nil || a.live == nil {
		return domain.PutCellMutationResult{}, fmt.Errorf("hexxlastore cell writer: nil database")
	}
	if cmd == nil {
		return domain.PutCellMutationResult{}, fmt.Errorf("hexxlastore cell writer: nil command")
	}
	kind := cmd.Kind
	if kind == "" {
		kind = domain.CellPutKindFact
	}
	placement := cmd.Placement
	if placement == "" {
		placement = domain.CellPlacementExact
	}
	result := domain.PutCellMutationResult{Placement: placement}
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		updErr := db.Update(func(tx *hexxladb.Tx) error {
			selected := hexxladb.Coord{Q: cmd.Coord.Q, R: cmd.Coord.R}
			var key hexxladb.PackedCoord
			switch placement {
			case domain.CellPlacementExact:
				var err error
				key, err = hexxladb.Pack(selected)
				if err != nil {
					return fmt.Errorf("pack exact coordinate: %w", err)
				}
				_, occupied, err := tx.GetCell(key)
				if err != nil {
					return fmt.Errorf("check exact coordinate: %w", err)
				}
				if occupied && !cmd.AllowOverwrite {
					return fmt.Errorf("%w: (%d,%d)", domain.ErrCellCoordinateOccupied, selected.Q, selected.R)
				}
				result.Replaced = occupied
			case domain.CellPlacementNearAnchor:
				placed, err := tx.FindFreeCellPlacement(ctx, selected, cmd.MaxRadius)
				if err != nil {
					return fmt.Errorf("find free cell placement: %w", err)
				}
				selected = placed.Coord
				key = placed.Key
				result.Probes = placed.Probes
			default:
				return fmt.Errorf("unknown cell placement %q", placement)
			}

			rec, err := newCellRecord(key, cmd, kind)
			if err != nil {
				return fmt.Errorf("build cell record: %w", err)
			}
			if err := tx.PutCell(ctx, rec); err != nil {
				return fmt.Errorf("tx PutCell: %w", err)
			}
			result.OK = true
			result.Coord = domain.AxialCoord{Q: selected.Q, R: selected.R}
			return nil
		})
		if updErr != nil {
			return fmt.Errorf("hexxlastore put cell: %w", updErr)
		}
		return nil
	})
	if err != nil {
		return domain.PutCellMutationResult{}, err
	}
	return result, nil
}

func newCellRecord(key hexxladb.PackedCoord, cmd *domain.PutCellCommand, kind domain.CellPutKind) (hexxladb.CellRecord, error) {
	var rec hexxladb.CellRecord
	switch kind {
	case domain.CellPutKindFact:
		rec = hexxladb.NewFactCell(key, cmd.RawContent, cmd.SourceID, "mcp-cell", cmd.Confidence)
	case domain.CellPutKindUserMessage:
		rec = hexxladb.NewUserMessageCell(key, cmd.RawContent, cmd.SourceID, cmd.Confidence)
	case domain.CellPutKindAssistantResponse:
		rec = hexxladb.NewAssistantResponseCell(key, cmd.RawContent, cmd.SourceID, cmd.Confidence)
	default:
		return hexxladb.CellRecord{}, fmt.Errorf("unknown cell kind %q", kind)
	}
	rec.Tags = mergeTagLists(rec.Tags, cmd.Tags)
	return rec, nil
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
			return fmt.Errorf("hexxlastore put embedding: database has no configured embedding dimension")
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
