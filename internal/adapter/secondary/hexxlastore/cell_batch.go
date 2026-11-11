package hexxlastore

import (
	"context"
	"errors"
	"fmt"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// PutCells shares placement, metadata, embedding and receipt semantics with PutCell.
func (a *CellWriterAdapter) PutCells(ctx context.Context, cmd *domain.PutCellsCommand, vectors [][]float32) (domain.PutCellsResult, error) {
	if cmd == nil || len(cmd.Cells) != len(vectors) {
		return domain.PutCellsResult{}, fmt.Errorf("batch cell/vector counts differ")
	}
	out := domain.PutCellsResult{Results: make([]domain.PutCellMutationResult, len(cmd.Cells))}
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		err := updateMutation(db, func(tx *hexxladb.Tx) error {
			changed := false
			for i := range cmd.Cells {
				err := a.putCellInTx(ctx, tx, &cmd.Cells[i], vectors[i], &out.Results[i])
				if errors.Is(err, errPutCellReplay) {
					continue
				}
				if err != nil {
					return fmt.Errorf("batch cell %d: %w", i, err)
				}
				changed = true
			}
			if !changed {
				return errPutCellReplay
			}
			return nil
		})
		if errors.Is(err, errPutCellReplay) {
			return nil
		}
		return err
	})
	if err != nil && !errors.Is(err, domain.ErrMutationCommitDurable) {
		return domain.PutCellsResult{}, err
	}
	out.OK = true
	return out, err
}
