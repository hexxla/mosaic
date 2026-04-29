package hexxlastore

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/ports/secondary"
)

// CellWriterAdapter implements [secondary.CellWriter] using (*hexxladb.DB).Update,
// [hexxladb.Pack], and HexxlaDB template helpers for stored cell records.
type CellWriterAdapter struct {
	db *hexxladb.DB
}

// NewCellWriterAdapter wraps an open database (caller owns Open/Close).
func NewCellWriterAdapter(db *hexxladb.DB) *CellWriterAdapter {
	return &CellWriterAdapter{db: db}
}

// PutCell implements [secondary.CellWriter].
func (a *CellWriterAdapter) PutCell(ctx context.Context, cmd *domain.PutCellCommand) error {
	if a == nil || a.db == nil {
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
	updErr := a.db.Update(func(tx *hexxladb.Tx) error {
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
}

// PutEmbedding implements [secondary.CellWriter].
func (a *CellWriterAdapter) PutEmbedding(ctx context.Context, coord domain.AxialCoord, vec []float32) error {
	if a == nil || a.db == nil {
		return fmt.Errorf("hexxlastore cell writer: nil database")
	}
	pk, err := hexxladb.Pack(hexxladb.Coord{Q: coord.Q, R: coord.R})
	if err != nil {
		return fmt.Errorf("hexxlastore put embedding: %w", err)
	}
	dim := a.db.EmbeddingDimension()
	if dim == 0 {
		return fmt.Errorf("hexxlastore put embedding: %w", hexxladb.ErrEmbeddingsDisabled)
	}
	if len(vec) != int(dim) {
		return fmt.Errorf("hexxlastore put embedding: vector length %d, want %d", len(vec), dim)
	}
	updErr := a.db.Update(func(tx *hexxladb.Tx) error {
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
}

// DeleteCell implements [secondary.CellWriter].
func (a *CellWriterAdapter) DeleteCell(ctx context.Context, cmd *domain.DeleteCellCommand) error {
	if a == nil || a.db == nil {
		return fmt.Errorf("hexxlastore cell writer: nil database")
	}
	if cmd == nil {
		return fmt.Errorf("hexxlastore cell writer: nil command")
	}
	pk, err := hexxladb.Pack(hexxladb.Coord{Q: cmd.Coord.Q, R: cmd.Coord.R})
	if err != nil {
		return fmt.Errorf("hexxlastore delete cell: %w", err)
	}
	updErr := a.db.Update(func(tx *hexxladb.Tx) error {
		if err := tx.DeleteCell(ctx, pk); err != nil {
			return fmt.Errorf("tx DeleteCell: %w", err)
		}
		return nil
	})
	if updErr != nil {
		return fmt.Errorf("hexxlastore delete cell: %w", updErr)
	}
	return nil
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
