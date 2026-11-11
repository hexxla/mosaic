package services

import (
	"context"
	"fmt"

	"github.com/hexxla/mosaic/internal/core/domain"
	"github.com/hexxla/mosaic/internal/core/ports/secondary"
)

// PutCells validates all commands and prepares model effects before one transaction.
func (s *CellMutationService) PutCells(ctx context.Context, cmd *domain.PutCellsCommand) (domain.PutCellsResult, error) {
	if cmd == nil || len(cmd.Cells) == 0 || len(cmd.Cells) > 32 {
		return domain.PutCellsResult{}, fmt.Errorf("put_cells requires 1..32 cells")
	}
	writer, ok := s.writer.(secondary.BatchCellWriter)
	if !ok {
		return domain.PutCellsResult{}, fmt.Errorf("atomic batch writes unavailable")
	}
	normalized := &domain.PutCellsCommand{Cells: make([]domain.PutCellCommand, len(cmd.Cells))}
	vectors := make([][]float32, len(cmd.Cells))
	keys := make(map[string]bool)
	totalBytes := 0
	for i := range cmd.Cells {
		c, err := normalizePutCell(&cmd.Cells[i])
		if err != nil {
			return domain.PutCellsResult{}, fmt.Errorf("batch cell %d: %w", i, err)
		}
		if c.Placement != domain.CellPlacementNearAnchor || keys[c.IdempotencyKey] {
			return domain.PutCellsResult{}, fmt.Errorf("each batch cell requires near_anchor placement and a distinct idempotency_key")
		}
		keys[c.IdempotencyKey] = true
		if err := s.policy.PutCellDenied(c.Kind); err != nil {
			return domain.PutCellsResult{}, err
		}
		totalBytes += len(c.RawContent)
		for _, tag := range c.Tags {
			totalBytes += len(tag)
		}
		if totalBytes > 1<<20 {
			return domain.PutCellsResult{}, fmt.Errorf("batch content and tags exceed 1 MiB")
		}
		normalized.Cells[i] = *c
	}
	for i, c := range normalized.Cells {
		if err := ctx.Err(); err != nil {
			return domain.PutCellsResult{}, err
		}
		if c.Embed {
			_, found, err := s.writer.LookupPutCell(ctx, &c)
			if err != nil {
				return domain.PutCellsResult{}, err
			}
			if found {
				continue
			}
			if s.embedText == nil || s.embedDim == 0 {
				return domain.PutCellsResult{}, fmt.Errorf("batch embedding requires configured compatible model")
			}
			vector, err := s.embedText.Embed(ctx, c.RawContent, int(s.embedDim))
			if err != nil {
				return domain.PutCellsResult{}, err
			}
			if len(vector) != int(s.embedDim) {
				return domain.PutCellsResult{}, fmt.Errorf("batch embedding dimension mismatch")
			}
			vectors[i] = vector
		}
	}
	return writer.PutCells(ctx, normalized, vectors)
}
