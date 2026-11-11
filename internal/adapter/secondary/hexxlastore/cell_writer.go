package hexxlastore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hexxla/hexxladb"

	"github.com/hexxla/mosaic/internal/config"
	"github.com/hexxla/mosaic/internal/core/domain"
	"github.com/hexxla/mosaic/internal/core/ports/secondary"
)

// CellWriterAdapter implements [secondary.CellWriter] using (*hexxladb.DB).Update,
// [hexxladb.Pack], and HexxlaDB template helpers for stored cell records.
type CellWriterAdapter struct {
	live *LiveDB

	maintain config.DeleteAutoMaintainConfig

	maintenanceGate       chan struct{}
	debounceTimer         *time.Timer
	maintenanceGeneration uint64
	maintenanceStopped    bool
}

// NewCellWriterAdapter wraps the live database and its optional pruning policy.
func NewCellWriterAdapter(live *LiveDB, maintain config.DeleteAutoMaintainConfig) *CellWriterAdapter {
	return &CellWriterAdapter{live: live, maintain: maintain, maintenanceGate: make(chan struct{}, 1)}
}

// PutCell implements [secondary.CellWriter].
func (a *CellWriterAdapter) PutCell(ctx context.Context, cmd *domain.PutCellCommand, embedding []float32) (domain.PutCellMutationResult, error) {
	if a == nil || a.live == nil || cmd == nil {
		return domain.PutCellMutationResult{}, fmt.Errorf("hexxlastore cell writer: missing database or command")
	}
	var result domain.PutCellMutationResult
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		err := updateMutation(db, func(tx *hexxladb.Tx) error { return a.putCellInTx(ctx, tx, cmd, embedding, &result) })
		if errors.Is(err, errPutCellReplay) {
			return nil
		}
		return err
	})
	if err != nil && !errors.Is(err, domain.ErrMutationCommitDurable) {
		return domain.PutCellMutationResult{}, err
	}
	return result, err
}

func (a *CellWriterAdapter) putCellInTx(ctx context.Context, tx *hexxladb.Tx, cmd *domain.PutCellCommand, embedding []float32, result *domain.PutCellMutationResult) error {

	kind := cmd.Kind
	if kind == "" {
		kind = domain.CellPutKindFact
	}
	placement := cmd.Placement
	if placement == "" {
		placement = domain.CellPlacementExact
	}
	keyed := placement == domain.CellPlacementNearAnchor
	if keyed != (cmd.IdempotencyKey != "") {
		return errors.New("hexxlastore put cell: idempotency_key is required for near_anchor and applies only to that placement")
	}
	var request putCellRequest
	if keyed {
		var err error
		request, err = newPutCellRequest(cmd)
		if err != nil {
			return err
		}
	}
	*result = domain.PutCellMutationResult{Placement: placement}

	if err := ctx.Err(); err != nil {
		return err
	}
	if keyed {
		saved, found, err := request.lookup(tx)
		if err != nil {
			return err
		}
		if found {
			*result = saved
			// Abort this no-op transaction without advancing CommitSeq.
			return errPutCellReplay
		}
	}
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
	if result.Replaced {
		rec.Provenance.CreatedAt = prior.Provenance.CreatedAt
		if cmd.Tags == nil && (cmd.Metadata == nil || cmd.Metadata.FactType == "") {
			rec.Tags = slices.Clone(prior.Tags)
		}
		rec.Validity = prior.Validity
		rec.ClusterHint = prior.ClusterHint
	}
	if err := applyCellMetadata(&rec, cmd); err != nil {
		return err
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
	if len(embedding) > 0 {
		if err := tx.PutEmbedding(key, embedding); err != nil {
			return fmt.Errorf("tx PutEmbedding with cell: %w", err)
		}
	}
	result.Embedded = len(embedding) > 0
	result.OK = true
	result.Coord = domain.AxialCoord{Q: selected.Q, R: selected.R}
	if keyed {
		return request.store(tx, *result)
	}
	return nil
}

func newCellRecord(key hexxladb.PackedCoord, cmd *domain.PutCellCommand, kind domain.CellPutKind) (hexxladb.CellRecord, error) {
	var rec hexxladb.CellRecord
	switch kind {
	case domain.CellPutKindFact:
		factType := "mcp-cell"
		if cmd.Metadata != nil && cmd.Metadata.FactType != "" {
			factType = cmd.Metadata.FactType
		}
		rec = hexxladb.NewFactCell(key, cmd.RawContent, cmd.SourceID, factType, cmd.Confidence)
	case domain.CellPutKindSystemPrompt:
		rec = hexxladb.NewSystemPromptCell(key, cmd.RawContent, cmd.SourceID)
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

func applyCellMetadata(rec *hexxladb.CellRecord, cmd *domain.PutCellCommand) error {
	m := cmd.Metadata
	if m == nil {
		return nil
	}
	if m.Replace || m.ClearValidity {
		rec.Validity = hexxladb.ValidityWire{}
	}
	if m.Replace || m.ClearClusterHint {
		rec.ClusterHint = nil
	}
	if m.ValidFrom != nil {
		rec.Validity.ValidFrom = new(m.ValidFrom.UnixNano())
	}
	if m.ValidTo != nil {
		rec.Validity.ValidTo = new(m.ValidTo.UnixNano())
	}
	if rec.Validity.ValidFrom != nil && rec.Validity.ValidTo != nil && *rec.Validity.ValidFrom >= *rec.Validity.ValidTo {
		return fmt.Errorf("cell validity: valid_from must precede valid_to")
	}
	if m.ClusterHint != nil {
		packed, err := hexxladb.Pack(hexxladb.Coord(*m.ClusterHint))
		if err != nil {
			return fmt.Errorf("cluster_hint: %w", err)
		}
		rec.ClusterHint = &packed
	}
	if m.CreatedAt != nil {
		rec.Provenance.CreatedAt = m.CreatedAt.UnixNano()
	}
	if m.UpdatedAt != nil {
		rec.Provenance.UpdatedAt = m.UpdatedAt.UnixNano()
	}
	if m.ExactTags {
		rec.Tags = slices.Clone(cmd.Tags)
	}
	return nil
}

// PutEmbedding implements [secondary.CellWriter].
func (a *CellWriterAdapter) PutEmbedding(ctx context.Context, coord domain.AxialCoord, vec []float32, expectedContentSHA256 string) error {
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
		updErr := updateMutation(db, func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context: %w", err)
			}
			rec, found, err := tx.GetCell(pk)
			if err != nil {
				return fmt.Errorf("check embedding cell: %w", err)
			}
			if !found {
				return fmt.Errorf("embedding needs a live cell at (%d,%d): %w", coord.Q, coord.R, hexxladb.ErrCellNotFound)
			}
			if expectedContentSHA256 != "" {
				hash := sha256.Sum256([]byte(rec.RawContent))
				if hex.EncodeToString(hash[:]) != expectedContentSHA256 {
					return fmt.Errorf("embedding cell at (%d,%d): %w", coord.Q, coord.R, domain.ErrCellContentChanged)
				}
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
		return updateMutation(db, func(tx *hexxladb.Tx) error {
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
