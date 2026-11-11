package main

import (
	"context"
	"encoding/json/v2"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hexxla/hexxladb"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/adapter/primary/mcpsrv"
	"github.com/sploitzberg/mosaic/internal/adapter/secondary/hexxlastore"
	"github.com/sploitzberg/mosaic/internal/config"
	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/services"
)

type scopedTransport struct {
	*mcp.InMemoryTransport
	id string
}
type scopedConnection struct {
	mcp.Connection
	id string
}

func (c scopedConnection) SessionID() string { return c.id }
func (t scopedTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	conn, err := t.InMemoryTransport.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return scopedConnection{Connection: conn, id: t.id}, nil
}

func decodeMCPResult[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()
	var out T
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestRegionalTransferAndBatchThroughMCP(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	db, err := hexxladb.Open(filepath.Join(t.TempDir(), "transfer.hexxla"), nil)
	if err != nil {
		t.Fatal(err)
	}
	live := hexxlastore.NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })
	if err := db.Update(func(tx *hexxladb.Tx) error {
		key, _ := hexxladb.Pack(hexxladb.Coord{})
		return tx.PutCell(ctx, hexxladb.CellRecord{Key: key, RawContent: strings.Repeat("large", 500)})
	}); err != nil {
		t.Fatal(err)
	}
	server := mcpsrv.NewServer("transfer-test", "test", nil)
	gate, err := loadRatchetGate(true, "", true, slog.Default(), nil)
	if err != nil {
		t.Fatal(err)
	}
	server.AddReceivingMiddleware(gate.Middleware())
	explorer := hexxlastore.NewMemoryExplorerAdapter(live)
	browse := services.NewMemoryBrowseService(explorer)
	writer := hexxlastore.NewCellWriterAdapter(live, config.DeleteAutoMaintainConfig{})
	seams := hexxlastore.NewSeamStoreAdapter(live)
	for _, register := range []func() error{
		func() error { return mcpsrv.RegisterBrowseTools(server, browse, nil) },
		func() error {
			return mcpsrv.RegisterCellBatchTool(server, services.NewCellMutationService(writer, nil, 0))
		},
		func() error {
			return mcpsrv.RegisterListTagsTool(server, services.NewTagCatalogService(hexxlastore.NewTagCatalogAdapter(live)), nil, nil)
		},
		func() error { return mcpsrv.RegisterSeamRecordTool(server, services.NewSeamRecordService(seams)) },
		func() error {
			return mcpsrv.RegisterSeamTools(server, services.NewSeamLifecycleService(seams), nil, nil)
		},
	} {
		if err := register(); err != nil {
			t.Fatal(err)
		}
	}
	connect := func(id string) *mcp.ClientSession {
		st, ct := mcp.NewInMemoryTransports()
		ss, err := server.Connect(ctx, scopedTransport{InMemoryTransport: st, id: id}, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = ss.Close() })
		client := mcp.NewClient(&mcp.Implementation{Name: id, Version: "test"}, nil)
		session, err := client.Connect(ctx, ct, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = session.Close() })
		return session
	}
	a, b := connect("transfer-a"), connect("transfer-b")
	call := func(session *mcp.ClientSession, name string, args any) *mcp.CallToolResult {
		t.Helper()
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "mosaic_hexxla_" + name, Arguments: args})
		if err != nil || result.IsError {
			var message string
			if result != nil {
				for _, content := range result.Content {
					if text, ok := content.(*mcp.TextContent); ok {
						message += text.Text
					}
				}
			}
			t.Fatalf("%s=%s,%v", name, message, err)
		}
		return result
	}
	page := decodeMCPResult[domain.BrowsePage](t, call(a, "browse_cells", map[string]any{"center": map[string]any{"q": 0, "r": 0}, "radius": 1, "max_cells": 1}))
	if page.NextCursor == "" || len(page.Cells) != 1 || page.Cells[0].RawContent != "" {
		t.Fatalf("browse page=%+v", page)
	}
	foreign, err := b.CallTool(ctx, &mcp.CallToolParams{Name: "mosaic_hexxla_browse_cells", Arguments: map[string]any{"cursor": page.NextCursor}})
	if err != nil || !foreign.IsError {
		t.Fatalf("foreign cursor accepted=%+v,%v", foreign, err)
	}
	next := decodeMCPResult[domain.BrowsePage](t, call(a, "browse_cells", map[string]any{"cursor": page.NextCursor}))
	retry := decodeMCPResult[domain.BrowsePage](t, call(a, "browse_cells", map[string]any{"cursor": page.NextCursor}))
	if next.Done != retry.Done || next.Probes != retry.Probes || next.SnapshotSeq != retry.SnapshotSeq {
		t.Fatal("cursor replay changed result")
	}
	exported := decodeMCPResult[domain.BrowsePage](t, call(a, "export_cells", map[string]any{"center": map[string]any{"q": 0, "r": 0}, "max_bytes": 1024}))
	if len(exported.DeferredCells) != 1 {
		t.Fatalf("oversized export was lost=%+v", exported)
	}
	chunk := decodeMCPResult[domain.CellChunkResult](t, call(a, "read_cell_chunk", map[string]any{"coord": map[string]any{"q": 0, "r": 0}, "snapshot": map[string]any{"seq": exported.SnapshotSeq}, "format": "record_json", "max_bytes": 100}))
	if !chunk.Found || chunk.Done || chunk.NextOffset != 100 {
		t.Fatalf("record chunk=%+v", chunk)
	}
	args := map[string]any{"cells": []any{
		map[string]any{"q": 10, "r": 0, "raw_content": "one", "source_id": "batch", "confidence": 1, "placement": "near_anchor", "idempotency_key": "batch-one"},
		map[string]any{"q": 10, "r": 0, "raw_content": "two", "source_id": "batch", "confidence": 1, "placement": "near_anchor", "idempotency_key": "batch-two"},
	}}
	denied, err := a.CallTool(ctx, &mcp.CallToolParams{Name: "mosaic_hexxla_put_cells", Arguments: args})
	if err != nil || !denied.IsError {
		t.Fatal("batch bypassed Ratchet")
	}
	call(a, "list_tags", nil)
	first := decodeMCPResult[domain.PutCellsResult](t, call(a, "put_cells", args))
	call(a, "list_tags", nil)
	replay := decodeMCPResult[domain.PutCellsResult](t, call(a, "put_cells", args))
	if len(first.Results) != 2 || first.Results[0].Coord != replay.Results[0].Coord || first.Results[1].Coord != replay.Results[1].Coord {
		t.Fatal("batch replay created duplicate identities")
	}
	call(a, "find_seams", map[string]any{"center_q": 10, "center_r": 0})
	call(a, "put_seam", map[string]any{"id": "01ARZ3NDEKTSV4RRFFQ69G5FAV", "cell_a": first.Results[0].Coord, "cell_b": first.Results[1].Coord, "seam_type": "custom", "source_id": "test"})
}
