package main

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/hexxla/hexxladb"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/adapter/primary/mcpsrv"
	"github.com/sploitzberg/mosaic/internal/adapter/secondary/hexxlastore"
	"github.com/sploitzberg/mosaic/internal/core/services"
)

func TestAdvancedExplorationThroughMCP(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	db, err := hexxladb.Open(filepath.Join(t.TempDir(), "explore.hexxla"), &hexxladb.Options{EmbeddingDimension: 2, ChangelogEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	live := hexxlastore.NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })
	if err := db.Update(func(tx *hexxladb.Tx) error {
		for i := range 3 {
			key, _ := hexxladb.Pack(hexxladb.Coord{Q: i})
			if err := tx.PutCell(ctx, hexxladb.NewFactCell(key, "memory", "fixture", "topic", 1)); err != nil {
				return err
			}
			if err := tx.PutEmbedding(key, []float32{1, float32(i)}); err != nil {
				return err
			}
		}
		return tx.LinkCells(hexxladb.Coord{}, hexxladb.Coord{Q: 1}, "requires", 1, hexxladb.ProvenanceWire{})
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.TagSnapshot("before"); err != nil {
		t.Fatal(err)
	}
	if err := db.Update(func(tx *hexxladb.Tx) error {
		key, _ := hexxladb.Pack(hexxladb.Coord{})
		return tx.PutCell(ctx, hexxladb.NewFactCell(key, "changed", "fixture", "topic", 1))
	}); err != nil {
		t.Fatal(err)
	}
	store := hexxlastore.NewMemoryExplorerAdapter(live)
	svc := services.NewMemoryExplorationService(store)
	server := mcpsrv.NewServer("explorer-test", "test", nil)
	gate, err := loadRatchetGate(true, "", true, slog.Default(), nil)
	if err != nil {
		t.Fatal(err)
	}
	server.AddReceivingMiddleware(gate.Middleware())
	if err := mcpsrv.RegisterExplorationTools(server, svc, services.NewEmbeddingMaintenanceService(store, true), nil); err != nil {
		t.Fatal(err)
	}
	st, ct := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, sessionTransport{st}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "explorer-test", Version: "test"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	denied, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "mosaic_hexxla_delete_embedding", Arguments: map[string]any{"q": 0, "r": 0}})
	if err != nil || !denied.IsError {
		t.Fatalf("unguarded vector delete=%+v,%v", denied, err)
	}
	tools, err := session.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 16 {
		t.Fatalf("tools=%v err=%v", tools, err)
	}
	for _, test := range []struct {
		name string
		args map[string]any
	}{
		{"walk_graph", map[string]any{"start": map[string]any{"q": 0, "r": 0}, "relation": "requires"}},
		{"find_path", map[string]any{"start": map[string]any{"q": 0, "r": 0}, "goal": map[string]any{"q": 1, "r": 0}, "relation": "requires"}},
		{"load_fov", map[string]any{"seeds": []any{map[string]any{"q": 0, "r": 0}}, "radius": 2}},
		{"load_voronoi", map[string]any{"seeds": []any{map[string]any{"q": 0, "r": 0}, map[string]any{"q": 2, "r": 0}}, "radius": 2}},
		{"ring_density", map[string]any{"seeds": []any{map[string]any{"q": 0, "r": 0}}, "radius": 2}},
		{"render_grid", map[string]any{"seeds": []any{map[string]any{"q": 0, "r": 0}}, "radius": 2}},
		{"tag_cooccurrences", map[string]any{"center": map[string]any{"q": 0, "r": 0}}},
		{"untagged_cells", map[string]any{"center": map[string]any{"q": 0, "r": 0}, "radius": 2}},
		{"list_snapshots", nil},
		{"get_cell_at", map[string]any{"coord": map[string]any{"q": 0, "r": 0}, "snapshot": map[string]any{"tag": "before"}}},
		{"snapshot_diff", map[string]any{"from_seq": 1, "to_seq": 2}},
		{"read_changes", map[string]any{}},
		{"get_embedding", map[string]any{"q": 0, "r": 0}},
		{"storage_diagnostics", nil},
		{"rebuild_embedding_index", map[string]any{}},
		{"delete_embedding", map[string]any{"q": 0, "r": 0}},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "mosaic_hexxla_" + test.name, Arguments: test.args})
			if err != nil || result.IsError {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}
