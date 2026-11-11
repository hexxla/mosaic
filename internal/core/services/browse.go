package services

import (
	"context"
	"fmt"

	"github.com/hexxla/mosaic/internal/core/domain"
	"github.com/hexxla/mosaic/internal/core/ports/secondary"
)

// MemoryBrowseService owns page and chunk work/output limits.
type MemoryBrowseService struct{ reader secondary.MemoryBrowser }

// NewMemoryBrowseService constructs bounded regional transfer.
func NewMemoryBrowseService(reader secondary.MemoryBrowser) *MemoryBrowseService {
	return &MemoryBrowseService{reader: reader}
}

// Browse scans at most one bounded page of a snapshot-pinned region.
func (s *MemoryBrowseService) Browse(ctx context.Context, q domain.BrowseQuery) (domain.BrowsePage, error) {
	if q.Radius < 0 || q.Radius > 128 || q.MaxCells < 0 || q.MaxCells > 100 || q.MaxProbes < 0 || q.MaxProbes > 2000 || q.MaxBytes < 0 || q.MaxBytes > 100000 || (q.MaxBytes > 0 && q.MaxBytes < 1024) {
		return domain.BrowsePage{}, fmt.Errorf("browse limits: radius 0..128, cells 0..100, probes 0..2000, bytes 1024..100000")
	}
	if q.MaxCells == 0 {
		q.MaxCells = 50
	}
	if q.MaxProbes == 0 {
		q.MaxProbes = 1000
	}
	if q.MaxBytes == 0 {
		q.MaxBytes = 64000
	}
	if q.Ring < 0 || q.Ring > q.Radius+1 || q.Offset < 0 || q.Offset >= max(1, 6*q.Ring) {
		return domain.BrowsePage{}, fmt.Errorf("invalid regional continuation")
	}
	return s.reader.Browse(ctx, q)
}

// ReadCellChunk transfers at most 8192 raw bytes per call.
func (s *MemoryBrowseService) ReadCellChunk(ctx context.Context, q domain.CellChunkQuery) (domain.CellChunkResult, error) {
	if err := validateSnapshot(q.Snapshot); err != nil {
		return domain.CellChunkResult{}, err
	}
	if q.Format != "" && q.Format != "content" && q.Format != "record_json" {
		return domain.CellChunkResult{}, fmt.Errorf("format must be content or record_json")
	}
	if q.Offset < 0 || q.MaxBytes < 0 || q.MaxBytes > 8192 {
		return domain.CellChunkResult{}, fmt.Errorf("chunk offset must be nonnegative; max_bytes at most 8192")
	}
	if q.MaxBytes == 0 {
		q.MaxBytes = 8192
	}
	return s.reader.ReadCellChunk(ctx, q)
}
