package mcpsrv

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
)

func (rb *RetrievalBudgetTracker) registerReadTool[In, Out any](server *mcp.Server, name, description string, fn func(context.Context, In) (Out, error)) error {
	return addTool(server, &mcp.Tool{Name: name, Description: description}, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		out, err := rb.RunBudgetedRead(req, func() (Out, error) { return fn(ctx, in) })
		return nil, out, err
	})
}

// RegisterExplorationTools exposes bounded advanced reads and vector maintenance.
func RegisterExplorationTools(server *mcp.Server, svc primary.MemoryExplorer, maintenance primary.EmbeddingMaintenance, budget *RetrievalBudgetTracker) error {
	return registerTools(
		func() error {
			return budget.registerReadTool(server, "mosaic_hexxla_walk_graph", "Explore directed relationships with node, hop and adjacency-row limits. limit_reached means more nodes existed than the result quota.", svc.WalkGraph)
		},
		func() error {
			return budget.registerReadTool(server, "mosaic_hexxla_find_path", "Find a native weighted directed path. max_nodes bounds expansions; max_edge_rows bounds adjacency work. Empty coords means no path.", svc.FindPath)
		},
		func() error {
			return budget.registerReadTool(server, "mosaic_hexxla_load_fov", "Load bounded field-of-view context from exactly one seed. Explicit blocked coordinates are stable occluders; first blocking cells remain visible.", svc.LoadFOV)
		},
		func() error {
			return budget.registerReadTool(server, "mosaic_hexxla_load_voronoi", "Partition context among 1..8 seeds, with at most 100 cells divided among regions. Uses native uniform-cost Voronoi semantics.", svc.LoadVoronoi)
		},
		func() error {
			return budget.registerReadTool(server, "mosaic_hexxla_ring_density", "Inspect occupancy and geometric capacity around one seed, radius at most 16.", svc.Density)
		},
		func() error {
			return budget.registerReadTool(server, "mosaic_hexxla_render_grid", "Render a textual hex neighborhood around one seed, radius at most 10.", svc.RenderGrid)
		},
		func() error {
			return budget.registerReadTool(server, "mosaic_hexxla_tag_cooccurrences", "Count tag pairs in at most 1000 sampled cells, at most 64 tags per cell. partial identifies sampling, scan or result truncation. radius=0 samples globally; positive radius limits the region.", svc.TagPairs)
		},
		func() error {
			return budget.registerReadTool(server, "mosaic_hexxla_untagged_cells", "Find cells with no tags in a bounded regional scan, radius at most 16. partial means response coordinates were truncated.", svc.Untagged)
		},
		func() error {
			return budget.registerReadTool(server, "mosaic_hexxla_list_snapshots", "List up to 500 snapshot names and the oldest retained sequence. Names do not prevent retention expiry.", func(ctx context.Context, _ struct{}) (domain.SnapshotLabels, error) { return svc.ListSnapshots(ctx) })
		},
		func() error {
			return budget.registerReadTool(server, "mosaic_hexxla_get_cell_at", "Read a historical cell using exactly one snapshot selector: seq, tag or RFC3339 at. Expired history is rejected, never substituted with current data.", svc.GetHistoricalCell)
		},
		func() error {
			return budget.registerReadTool(server, "mosaic_hexxla_snapshot_diff", "Inspect retained cell/seam mutations across at most 64 commits, returning at most 100 changes. Native collection materializes the window under a five-second cooperative deadline. Not complete CDC and no continuation; narrow a truncated window.", svc.DiffHistory)
		},
		func() error {
			return budget.registerReadTool(server, "mosaic_hexxla_read_changes", "Peek at up to 128 changefeed records (32 with inline payloads), then filter operations. next_seq advances across examined records even with no matches. At-least-once delivery; opaque encoded keys/payloads are base64; large records may be hash-only. Does not acknowledge a consumer.", svc.ReadChanges)
		},
		func() error {
			return budget.registerReadTool(server, "mosaic_hexxla_get_embedding", "Read one current vector and recorded model identity without calling Ollama.", svc.GetVector)
		},
		func() error {
			return budget.registerReadTool(server, "mosaic_hexxla_storage_diagnostics", "Inspect native physical/live/reusable storage, cumulative write timings in nanoseconds, WAL counters and retention floor. Storage accounting walks the tree and can delay writers.", func(ctx context.Context, _ struct{}) (domain.DiagnosticsResult, error) { return svc.Diagnostics(ctx) })
		},
		func() error {
			return addTool(server, &mcp.Tool{Name: "mosaic_hexxla_delete_embedding", Description: "Delete a vector while retaining its cell. Requires a prior cell/vector inspection through Ratchet; disabled in lexical-only mode."}, func(ctx context.Context, _ *mcp.CallToolRequest, in domain.AxialCoord) (*mcp.CallToolResult, domain.MutationOK, error) {
				out, err := maintenance.DeleteVector(ctx, in)
				return nil, out, err
			})
		},
		func() error {
			return addTool(server, &mcp.Tool{Name: "mosaic_hexxla_rebuild_embedding_index", Description: "Rebuild only the derived ANN index, preserving authoritative vectors and model identity. Defaults: 1000 vectors and 128 MiB; ceilings: 10000 and 512 MiB. Concurrent vector changes reject publication safely."}, func(ctx context.Context, _ *mcp.CallToolRequest, in domain.RebuildIndexQuery) (*mcp.CallToolResult, domain.RebuildIndexResult, error) {
				out, err := maintenance.RebuildIndex(ctx, in)
				return nil, out, err
			})
		},
	)
}
