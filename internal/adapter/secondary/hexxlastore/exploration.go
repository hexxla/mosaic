package hexxlastore

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"

	"github.com/hexxla/hexxladb"

	"github.com/hexxla/mosaic/internal/core/domain"
)

// MemoryExplorerAdapter maps advanced workflows to public HexxlaDB operations.
type MemoryExplorerAdapter struct{ live *LiveDB }

// NewMemoryExplorerAdapter uses the command-owned database lifetime.
func NewMemoryExplorerAdapter(live *LiveDB) *MemoryExplorerAdapter {
	return &MemoryExplorerAdapter{live: live}
}

func (a *MemoryExplorerAdapter) read(ctx context.Context, fn func(*hexxladb.Tx) error) error {
	return a.live.WithRead(func(db *hexxladb.DB) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return db.View(func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return fn(tx)
		})
	})
}

// WalkGraph applies both node and physical adjacency-row budgets.
func (a *MemoryExplorerAdapter) WalkGraph(ctx context.Context, q domain.GraphQuery) (domain.CoordinateResults, error) {
	out := domain.CoordinateResults{Coords: []domain.AxialCoord{q.Start}}
	err := a.read(ctx, func(tx *hexxladb.Tx) error {
		seen := map[domain.AxialCoord]bool{q.Start: true}
		frontier := []domain.AxialCoord{q.Start}
		rows := 0
		for range q.MaxHops {
			var next []domain.AxialCoord
			for _, from := range frontier {
				key, err := hexxladb.Pack(hexxladb.Coord(from))
				if err != nil {
					return err
				}
				var visitErr error
				err = tx.AscendEdgesFrom(key, func(rec hexxladb.EdgeWalkRecord) bool {
					if visitErr = ctx.Err(); visitErr != nil {
						return false
					}
					rows++
					if rows > q.MaxEdgeRows {
						visitErr = hexxladb.ErrSpatialScanLimit
						return false
					}
					if q.Relation != "" && rec.RelationType != q.Relation {
						return true
					}
					coord, err := hexxladb.Unpack(rec.To)
					if err != nil {
						visitErr = err
						return false
					}
					c := domain.AxialCoord(coord)
					if seen[c] {
						return true
					}
					if len(out.Coords) >= q.MaxNodes {
						out.LimitReached = true
						return false
					}
					seen[c] = true
					out.Coords = append(out.Coords, c)
					next = append(next, c)
					return true
				})
				if err != nil {
					return err
				}
				if visitErr != nil {
					return visitErr
				}
				if out.LimitReached {
					return nil
				}
			}
			frontier = next
			if len(frontier) == 0 {
				break
			}
		}
		return nil
	})
	return out, err
}

// FindPath delegates native weighted pathfinding with work limits.
func (a *MemoryExplorerAdapter) FindPath(ctx context.Context, q domain.GraphQuery) (domain.CoordinateResults, error) {
	out := domain.CoordinateResults{Coords: []domain.AxialCoord{}}
	err := a.read(ctx, func(tx *hexxladb.Tx) error {
		coords, err := tx.FindEdgePath(ctx, hexxladb.Coord(q.Start), hexxladb.Coord(*q.Goal), hexxladb.FindEdgePathConfig{Filter: q.Relation, MaxExpand: q.MaxNodes, MaxEdgeRows: q.MaxEdgeRows})
		for _, c := range coords {
			out.Coords = append(out.Coords, domain.AxialCoord(c))
		}
		return err
	})
	return out, err
}

func recordHit(rec hexxladb.CellRecord) (domain.CellHit, error) {
	c, err := hexxladb.Unpack(rec.Key)
	if err != nil {
		return domain.CellHit{}, err
	}
	v := hexxladb.CellView{Coord: c, RawContent: rec.RawContent, Tags: rec.Tags, Provenance: rec.Provenance, Validity: rec.Validity}
	if rec.ClusterHint != nil {
		hint, err := hexxladb.Unpack(*rec.ClusterHint)
		if err != nil {
			return domain.CellHit{}, err
		}
		v.ClusterHint = &hint
	}
	return viewToCellHit(&v, 0, ""), nil
}
func recordHits(recs []hexxladb.CellRecord) ([]domain.CellHit, error) {
	out := make([]domain.CellHit, 0, len(recs))
	for _, rec := range recs {
		hit, err := recordHit(rec)
		if err != nil {
			return nil, err
		}
		out = append(out, hit)
	}
	return out, nil
}

// LoadFOV uses stable explicit blockers instead of remote callbacks.
func (a *MemoryExplorerAdapter) LoadFOV(ctx context.Context, q domain.SpatialQuery) (domain.SpatialResults, error) {
	out := domain.SpatialResults{Regions: []domain.SpatialRegion{}}
	err := a.read(ctx, func(tx *hexxladb.Tx) error {
		blocked := make(map[hexxladb.Coord]bool, len(q.Blocked))
		for _, c := range q.Blocked {
			coord := hexxladb.Coord(c)
			if _, err := hexxladb.Pack(coord); err != nil {
				return err
			}
			blocked[coord] = true
		}
		recs, err := tx.LoadContextFOV(ctx, hexxladb.Coord(q.Seeds[0]), q.Radius, func(c hexxladb.Coord) bool { return blocked[c] }, hexxladb.FOVContextConfig{MaxCells: q.MaxCells})
		if err != nil {
			return err
		}
		hits, err := recordHits(recs)
		out.Regions = append(out.Regions, domain.SpatialRegion{SeedIndex: 0, Cells: hits})
		return err
	})
	return out, err
}

// LoadVoronoi divides a bounded result quota among seeds deterministically.
func (a *MemoryExplorerAdapter) LoadVoronoi(ctx context.Context, q domain.SpatialQuery) (domain.SpatialResults, error) {
	out := domain.SpatialResults{Regions: []domain.SpatialRegion{}}
	err := a.read(ctx, func(tx *hexxladb.Tx) error {
		seeds := make([]hexxladb.Coord, len(q.Seeds))
		for i, c := range q.Seeds {
			seeds[i] = hexxladb.Coord(c)
		}
		regions, err := tx.LoadContextVoronoi(ctx, seeds, hexxladb.VoronoiContextConfig{MaxRadius: q.Radius, MaxCellsPerSeed: max(1, q.MaxCells/len(seeds))})
		if err != nil {
			return err
		}
		remaining := q.MaxCells
		for i := range seeds {
			recs := regions[i]
			recs = recs[:min(len(recs), remaining)]
			hits, err := recordHits(recs)
			if err != nil {
				return err
			}
			out.Regions = append(out.Regions, domain.SpatialRegion{SeedIndex: i, Cells: hits})
			remaining -= len(hits)
		}
		return nil
	})
	return out, err
}

// Density delegates bounded native occupancy enumeration.
func (a *MemoryExplorerAdapter) Density(ctx context.Context, q domain.SpatialQuery) (domain.DensityResults, error) {
	out := domain.DensityResults{Rings: []domain.RingOccupancy{}}
	err := a.read(ctx, func(tx *hexxladb.Tx) error {
		rings, err := tx.RingDensityMap(ctx, hexxladb.Coord(q.Seeds[0]), q.Radius)
		for _, r := range rings {
			out.Rings = append(out.Rings, domain.RingOccupancy{Ring: r.Ring, Occupied: r.Occupied, Total: r.Total})
		}
		return err
	})
	return out, err
}

// RenderGrid delegates bounded native text rendering.
func (a *MemoryExplorerAdapter) RenderGrid(ctx context.Context, q domain.SpatialQuery) (domain.GridResult, error) {
	var out domain.GridResult
	err := a.read(ctx, func(tx *hexxladb.Tx) error {
		var err error
		out.Grid, err = tx.RenderHexGridFromDB(ctx, hexxladb.Coord(q.Seeds[0]), q.Radius)
		return err
	})
	return out, err
}

// TagPairs computes pair counts over a bounded, explicitly reported sample.
func (a *MemoryExplorerAdapter) TagPairs(ctx context.Context, q domain.TaxonomyQuery) (domain.TaxonomyResults, error) {
	out := domain.TaxonomyResults{Pairs: []domain.TagPairCount{}}
	err := a.read(ctx, func(tx *hexxladb.Tx) error {
		hits, err := tx.QueryCells(ctx, hexxladb.CellQuery{Center: hexxladb.Coord(q.Center), Radius: q.Radius, SortBy: hexxladb.SortByCoord, MaxScanRows: q.MaxScanRows, MaxResults: 1001})
		out.Partial = errors.Is(err, hexxladb.ErrQueryScanLimit)
		if err != nil && !out.Partial {
			return err
		}
		if len(hits) > 1000 {
			hits = hits[:1000]
			out.Partial = true
		}
		out.CellsExamined = len(hits)
		counts := make(map[[2]string]int)
		for _, hit := range hits {
			if err := ctx.Err(); err != nil {
				return err
			}
			tags := slices.Clone(hit.Cell.Tags)
			slices.Sort(tags)
			tags = slices.Compact(tags)
			if len(tags) > 64 {
				tags = tags[:64]
				out.Partial = true
			}
			for i, a := range tags {
				for _, b := range tags[i+1:] {
					pair := [2]string{a, b}
					if _, exists := counts[pair]; !exists && len(counts) >= 10000 {
						out.Partial = true
						continue
					}
					counts[pair]++
				}
			}
		}
		for pair, count := range counts {
			if count >= q.MinCount {
				out.Pairs = append(out.Pairs, domain.TagPairCount{A: pair[0], B: pair[1], Count: count})
			}
		}
		slices.SortFunc(out.Pairs, func(a, b domain.TagPairCount) int {
			return cmp.Or(cmp.Compare(b.Count, a.Count), cmp.Compare(a.A, b.A), cmp.Compare(a.B, b.B))
		})
		if len(out.Pairs) > q.MaxResults {
			out.Pairs = out.Pairs[:q.MaxResults]
			out.Partial = true
		}
		return nil
	})
	return out, err
}

// Untagged inspects only the bounded region and caps response coordinates.
func (a *MemoryExplorerAdapter) Untagged(ctx context.Context, q domain.TaxonomyQuery) (domain.TaxonomyResults, error) {
	out := domain.TaxonomyResults{Coords: []domain.AxialCoord{}}
	err := a.read(ctx, func(tx *hexxladb.Tx) error {
		for i, c := range hexxladb.WalkRings(nil, hexxladb.Coord(q.Center), q.Radius) {
			if err := ctx.Err(); err != nil {
				return err
			}
			if i >= q.MaxScanRows {
				out.Partial = true
				break
			}
			key, err := hexxladb.Pack(c)
			if err != nil {
				return err
			}
			rec, found, err := tx.GetCell(key)
			if err != nil {
				return err
			}
			if !found {
				continue
			}
			out.CellsExamined++
			if len(rec.Tags) == 0 {
				if len(out.Coords) >= q.MaxResults {
					out.Partial = true
					break
				}
				out.Coords = append(out.Coords, domain.AxialCoord(c))
			}
		}
		return nil
	})
	return out, err
}

func viewSnapshot(db *hexxladb.DB, sel *domain.SnapshotSelector, fn func(*hexxladb.Tx) error) error {
	if sel == nil {
		return db.View(fn)
	}
	if sel.Seq != nil {
		return db.ViewAt(*sel.Seq, fn)
	}
	if sel.Tag != "" {
		return db.ViewAtTag(sel.Tag, fn)
	}
	if sel.At != nil {
		return db.ViewAtTime(*sel.At, fn)
	}
	return fmt.Errorf("empty snapshot selector")
}

// ListSnapshots reports a bounded list and the native retention boundary.
func (a *MemoryExplorerAdapter) ListSnapshots(ctx context.Context) (domain.SnapshotLabels, error) {
	out := domain.SnapshotLabels{Tags: []domain.SnapshotLabel{}}
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		tags, err := db.ListSnapshotTags()
		if err != nil {
			return err
		}
		out.Truncated = len(tags) > 500
		for _, tag := range tags[:min(500, len(tags))] {
			out.Tags = append(out.Tags, domain.SnapshotLabel{Label: tag.Label, Seq: tag.CommitSeq})
		}
		out.OldestSeq, err = db.OldestSnapshotSeq()
		return err
	})
	return out, err
}

// GetHistoricalCell never mixes current graph or vector records into old cells.
func (a *MemoryExplorerAdapter) GetHistoricalCell(ctx context.Context, q domain.HistoricalCellQuery) (domain.CellLookupResponse, error) {
	var out domain.CellLookupResponse
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
			hit, err := recordHit(rec)
			out = domain.CellLookupResponse{Found: true, Cell: &hit}
			return err
		})
	})
	return out, err
}

func seamHit(rec hexxladb.SeamRecord) (domain.SeamHit, error) {
	a, err := hexxladb.Unpack(rec.CellA)
	if err != nil {
		return domain.SeamHit{}, err
	}
	b, err := hexxladb.Unpack(rec.CellB)
	if err != nil {
		return domain.SeamHit{}, err
	}
	return domain.SeamHit{Confidence: rec.Provenance.Confidence, CreatedAt: nanoWallToRFC3339(rec.Provenance.CreatedAt), UpdatedAt: nanoWallToRFC3339(rec.Provenance.UpdatedAt), ValidFrom: nanoWallPtrToRFC3339(rec.Validity.ValidFrom), ValidTo: nanoWallPtrToRFC3339(rec.Validity.ValidTo), ID: rec.ID, CellA: domain.AxialCoord(a), CellB: domain.AxialCoord(b), SeamType: rec.SeamType, Reason: rec.Reason, ConfidenceDelta: rec.ConfidenceDelta, DetectedAtRFC3339: nanoWallToRFC3339(rec.DetectedAt), ResolutionStatus: rec.ResolutionStatus, ResolutionNote: rec.ResolutionNote, SourceID: rec.Provenance.SourceID}, nil
}

// DiffHistory exposes retained diagnostic changes with an explicit response cap.
func (a *MemoryExplorerAdapter) DiffHistory(ctx context.Context, q domain.HistoryDiffQuery) (domain.HistoryDiffResults, error) {
	out := domain.HistoryDiffResults{Changes: []domain.HistoryChange{}}
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		diff, err := db.SnapshotDiff(ctx, q.FromSeq, q.ToSeq, hexxladb.SnapshotDiffConfig{})
		if err != nil {
			return err
		}
		out.OldestSeq, err = db.OldestSnapshotSeq()
		if err != nil {
			return err
		}
		out.HistoryPruned = q.FromSeq < out.OldestSeq
		for _, c := range diff.Cells {
			hit, err := recordHit(c.Record)
			if err != nil {
				return err
			}
			hit.Coord = domain.AxialCoord(c.Coord)
			out.Changes = append(out.Changes, domain.HistoryChange{Seq: c.CommitSeq, Operation: string(c.Op), Cell: &hit})
		}
		for _, s := range diff.Seams {
			hit, err := seamHit(s.Record)
			if err != nil {
				return err
			}
			out.Changes = append(out.Changes, domain.HistoryChange{Seq: s.CommitSeq, Operation: string(s.Op), Seam: &hit})
		}
		slices.SortStableFunc(out.Changes, func(a, b domain.HistoryChange) int { return cmp.Compare(a.Seq, b.Seq) })
		out.Truncated = len(out.Changes) > q.MaxResults
		out.Changes = out.Changes[:min(len(out.Changes), q.MaxResults)]
		return nil
	})
	return out, err
}

func changeOperation(op byte) string {
	switch op {
	case hexxladb.ChangelogOpPutCell:
		return "put_cell"
	case hexxladb.ChangelogOpDeleteCell:
		return "delete_cell"
	case hexxladb.ChangelogOpPutFacet:
		return "put_facet"
	case hexxladb.ChangelogOpPutEdge:
		return "put_edge"
	case hexxladb.ChangelogOpPutSeam:
		return "put_seam"
	case hexxladb.ChangelogOpResolveSeam:
		return "resolve_seam"
	default:
		return fmt.Sprintf("unknown_%d", op)
	}
}

// ReadChanges filters after a bounded physical record read to guarantee progress.
func (a *MemoryExplorerAdapter) ReadChanges(ctx context.Context, q domain.ChangesQuery) (domain.ChangesResults, error) {
	out := domain.ChangesResults{Events: []domain.ChangeEvent{}, NextSeq: q.AfterSeq}
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		records, err := db.ReadChangelogSince(q.AfterSeq, q.MaxRecords)
		if err != nil {
			return err
		}
		out.MayHaveMore = len(records) == q.MaxRecords
		for _, r := range records {
			if err := ctx.Err(); err != nil {
				return err
			}
			out.NextSeq = r.Seq
			op := changeOperation(r.Op)
			if len(q.Operations) > 0 && !slices.Contains(q.Operations, op) {
				continue
			}
			event := domain.ChangeEvent{Seq: r.Seq, Operation: op, WallTime: nanoWallToRFC3339(r.WallUnixNs), KeyBase64: base64.StdEncoding.EncodeToString(r.Key), EncodedBytes: r.EncodedLen}
			if r.HashValid {
				event.SHA256 = hex.EncodeToString(r.Hash[:])
			}
			if q.IncludePayload {
				event.PayloadBase64 = base64.StdEncoding.EncodeToString(r.Inline)
			}
			out.Events = append(out.Events, event)
		}
		return nil
	})
	return out, err
}

// GetVector reports current vector data and application-owned model identity.
func (a *MemoryExplorerAdapter) GetVector(ctx context.Context, q domain.AxialCoord) (domain.VectorResult, error) {
	var out domain.VectorResult
	err := a.read(ctx, func(tx *hexxladb.Tx) error {
		key, err := hexxladb.Pack(hexxladb.Coord(q))
		if err != nil {
			return err
		}
		out.Vector, out.Found, err = tx.GetEmbedding(key)
		if err != nil {
			return err
		}
		model, _, err := tx.Get([]byte(embeddingModelKey))
		out.Model = string(model)
		return err
	})
	return out, err
}

// DeleteVector preserves the cell and classifies commit finalization failures.
func (a *MemoryExplorerAdapter) DeleteVector(ctx context.Context, q domain.AxialCoord) (domain.MutationOK, error) {
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		return updateMutation(db, func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			key, err := hexxladb.Pack(hexxladb.Coord(q))
			if err != nil {
				return err
			}
			return tx.DeleteEmbedding(key)
		})
	})
	return domain.MutationOK{OK: err == nil}, err
}

// RebuildIndex bounds native in-memory construction and publication.
func (a *MemoryExplorerAdapter) RebuildIndex(ctx context.Context, q domain.RebuildIndexQuery) (domain.RebuildIndexResult, error) {
	var out domain.RebuildIndexResult
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		stats, err := db.RebuildEmbeddingIndex(ctx, &hexxladb.EmbeddingIndexRebuildOptions{MaxVectors: q.MaxVectors, MaxMemoryBytes: q.MaxMemoryBytes})
		out = domain.RebuildIndexResult{Vectors: stats.Vectors, Revision: stats.Revision, EstimatedPeakBytes: stats.EstimatedPeakBytes}
		return classifyMutationCommit(err)
	})
	return out, err
}

// Diagnostics uses typed public accounting rather than decoding physical pages.
func (a *MemoryExplorerAdapter) Diagnostics(ctx context.Context) (domain.DiagnosticsResult, error) {
	var out domain.DiagnosticsResult
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		storage, err := db.StorageStats()
		if err != nil {
			return err
		}
		out.Storage = map[string]uint64{"page_size": storage.PageSize, "physical_page_size": storage.PhysicalPageSize, "primary_bytes": storage.PrimaryBytes, "wal_bytes": storage.WALBytes, "changelog_bytes": storage.ChangelogBytes, "allocated_pages": storage.AllocatedPages, "reachable_pages": storage.ReachablePages, "allocator_pages": storage.AllocatorPages, "reusable_pages": storage.ReusablePages, "live_bytes": storage.LiveBytes, "reclaimable_bytes": storage.ReclaimableBytes}
		w := db.WriteStats()
		attempts, syncs := db.WALStats()
		out.Writes = map[string]uint64{"calls": w.Calls, "commits": w.Commits, "lock_wait_ns": uint64(max(0, w.LockWait)), "callback_ns": uint64(max(0, w.Callback)), "durability_ns": uint64(max(0, w.Durability)), "finalization_ns": uint64(max(0, w.Finalization)), "wal_commit_attempts": attempts, "redo_syncs": syncs}
		out.OldestSeq, err = db.OldestSnapshotSeq()
		if err != nil {
			return err
		}
		return db.View(func(tx *hexxladb.Tx) error {
			model, _, err := tx.Get([]byte(embeddingModelKey))
			out.Model = string(model)
			return err
		})
	})
	return out, err
}
