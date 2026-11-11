package secondary

import (
	"context"

	"github.com/hexxla/mosaic/internal/core/domain"
)

// MemoryBrowser reads deterministic regional pages and snapshot-pinned byte chunks.
type MemoryBrowser interface {
	Browse(context.Context, domain.BrowseQuery) (domain.BrowsePage, error)
	ReadCellChunk(context.Context, domain.CellChunkQuery) (domain.CellChunkResult, error)
}
