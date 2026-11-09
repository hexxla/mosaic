package hexxlastore

import (
	"context"
	"errors"
	"fmt"
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
	live *LiveDB

	maintain config.DeleteAutoMaintainConfig

	maintainMu            sync.Mutex
	debounceTimer         *time.Timer
	maintenanceGeneration uint64
	maintenanceStopped    bool
}

// NewCellWriterAdapter wraps the live database and its optional pruning policy.
func NewCellWriterAdapter(live *LiveDB, maintain config.DeleteAutoMaintainConfig) *CellWriterAdapter {
	return &CellWriterAdapter{live: live, maintain: maintain}
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
			var prior hexxladb.CellRecord
			switch placement {
			case domain.CellPlacementExact:
				var err error
				key, err = hexxladb.Pack(selected)
				if err != nil {
					return fmt.Errorf("pack exact coordinate: %w", err)
				}
				previous, occupied, err := tx.GetCell(key)
				if err != nil {
					return fmt.Errorf("check exact coordinate: %w", err)
				}
				if occupied && !cmd.AllowOverwrite {
					return fmt.Errorf("%w: (%d,%d)", domain.ErrCellCoordinateOccupied, selected.Q, selected.R)
				}
				if !occupied {
					if _, err := tx.FindUnusedCellPlacement(ctx, selected, 0); err != nil {
						if errors.Is(err, hexxladb.ErrNoFreeCellPlacement) {
							return fmt.Errorf("%w: (%d,%d) has retained cell history", domain.ErrCellCoordinateOccupied, selected.Q, selected.R)
						}
						return fmt.Errorf("check unused coordinate: %w", err)
					}
				}
				prior = previous
				result.Replaced = occupied
			case domain.CellPlacementNearAnchor:
				placed, err := tx.FindUnusedCellPlacement(ctx, selected, cmd.MaxRadius)
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
			// New cells can encounter orphan vectors written by older versions.
			if !result.Replaced || prior.RawContent != rec.RawContent {
				_, found, err := tx.GetEmbedding(key)
				if err != nil {
					return fmt.Errorf("check previous embedding: %w", err)
				}
				if found {
					if err := tx.DeleteEmbedding(key); err != nil {
						return fmt.Errorf("invalidate previous embedding: %w", err)
					}
				}
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
			if _, found, err := tx.GetCell(pk); err != nil {
				return fmt.Errorf("check embedding cell: %w", err)
			} else if !found {
				return fmt.Errorf("embedding needs a live cell at (%d,%d): %w", coord.Q, coord.R, hexxladb.ErrCellNotFound)
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
			var err error
			removed, err = tx.DeleteCellWithOutcome(ctx, pk)
			return err
		})
	})
	if delErr != nil {
		return false, fmt.Errorf("hexxlastore delete cell: %w", delErr)
	}
	if removed && a.maintain.Enabled && a.maintain.Prune {
		a.scheduleMaintenance(ctx)
	}
	return removed, nil
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
