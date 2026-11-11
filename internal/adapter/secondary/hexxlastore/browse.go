package hexxlastore

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json/v2"
	"fmt"

	"github.com/hexxla/hexxladb"

	"github.com/hexxla/mosaic/internal/core/domain"
)

// Browse pins a sequence, then resumes by geometric ring/offset without storing results.
func (a *MemoryExplorerAdapter) Browse(ctx context.Context, q domain.BrowseQuery) (domain.BrowsePage, error) {
	out := domain.BrowsePage{Cells: []domain.CellHit{}, NextRing: q.Ring, NextOffset: q.Offset}
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if q.SnapshotSeq == nil {
			var sequence uint64
			currentAPI := false
			if err := db.View(func(tx *hexxladb.Tx) error {
				if snapshot, ok := any(tx).(interface{ ReadSequence() uint64 }); ok {
					sequence = snapshot.ReadSequence()
					currentAPI = true
				}
				return nil
			}); err != nil {
				return err
			}
			if currentAPI {
				out.SnapshotSeq = sequence
			} else {
				stats, err := db.StatsMVCC()
				if err != nil {
					return err
				}
				out.SnapshotSeq = stats.CommitSeq
			}
		} else {
			out.SnapshotSeq = *q.SnapshotSeq
		}
		return db.ViewAt(out.SnapshotSeq, func(tx *hexxladb.Tx) error {
			usedBytes := 0
			stopped := false

			for ring := q.Ring; ring <= q.Radius; ring++ {
				coords := hexxladb.Ring(hexxladb.Coord(q.Center), ring)
				start := 0
				if ring == q.Ring {
					start = q.Offset
				}
				for offset := start; offset < len(coords); offset++ {
					if err := ctx.Err(); err != nil {
						return err
					}
					if out.Probes >= q.MaxProbes || len(out.Cells)+len(out.DeferredCells) >= q.MaxCells {
						out.NextRing = ring
						out.NextOffset = offset
						stopped = true
						break
					}
					coord := coords[offset]
					out.Probes++
					out.NextRing = ring
					out.NextOffset = offset + 1
					if offset+1 == len(coords) {
						out.NextRing = ring + 1
						out.NextOffset = 0
					}
					key, err := hexxladb.Pack(coord)
					if err != nil {
						return err
					}
					rec, exists, err := tx.GetCell(key)
					if err != nil {
						return err
					}
					if !exists {
						continue
					}
					hit, err := recordHit(rec)
					if err != nil {
						return err
					}
					if !q.IncludeContent {
						hit.RawContent = ""
					}
					encoded, err := json.Marshal(hit)
					if err != nil {
						return err
					}
					if len(encoded) > q.MaxBytes {
						out.DeferredCells = append(out.DeferredCells, domain.AxialCoord(coord))
						continue
					}
					if usedBytes+len(encoded) > q.MaxBytes {
						out.NextRing = ring
						out.NextOffset = offset
						stopped = true
						break
					}
					usedBytes += len(encoded)
					out.Cells = append(out.Cells, hit)
				}
				if stopped {
					break
				}
				out.NextRing = ring + 1
				out.NextOffset = 0
			}
			out.Done = out.NextRing > q.Radius
			return nil
		})
	})
	return out, err
}

// ReadCellChunk preserves arbitrary content bytes and complete native record fields.
func (a *MemoryExplorerAdapter) ReadCellChunk(ctx context.Context, q domain.CellChunkQuery) (domain.CellChunkResult, error) {
	var out domain.CellChunkResult
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		return viewSnapshot(db, &q.Snapshot, func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			key, err := hexxladb.Pack(hexxladb.Coord(q.Coord))
			if err != nil {
				return err
			}
			rec, found, err := tx.GetCell(key)
			if err != nil || !found {
				return err
			}
			data := []byte(rec.RawContent)
			if q.Format == "record_json" {
				data, err = json.Marshal(rec)
				if err != nil {
					return err
				}
			}
			if q.Offset > len(data) {
				return fmt.Errorf("chunk offset exceeds record length")
			}
			out.Found = true
			out.TotalBytes = len(data)
			out.NextOffset = q.Offset + min(q.MaxBytes, len(data)-q.Offset)
			hash := sha256.Sum256(data)
			out.SHA256 = hex.EncodeToString(hash[:])
			out.DataBase64 = base64.StdEncoding.EncodeToString(data[q.Offset:out.NextOffset])
			out.Done = out.NextOffset == len(data)
			return nil
		})
	})
	return out, err
}
