package main

import (
	"context"
	"encoding/json/v2"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hexxla/hexxladb"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hexxla/mosaic/internal/adapter/primary/mcpsrv"
	"github.com/hexxla/mosaic/internal/adapter/secondary/hexxlastore"
	"github.com/hexxla/mosaic/internal/config"
	"github.com/hexxla/mosaic/internal/core/domain"
	"github.com/hexxla/mosaic/internal/core/ports/secondary"
	"github.com/hexxla/mosaic/internal/core/services"
)

func TestNearAnchorRetryAfterLostMCPResponse(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	path := filepath.Join(t.TempDir(), "memory.hexxla")
	db, err := hexxladb.Open(path, &hexxladb.Options{ChangelogEnabled: true, EmbeddingDimension: 2})
	if err != nil {
		t.Fatal(err)
	}
	live := hexxlastore.NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })
	var drop atomic.Bool
	first := connectMutationRetryClient(t, ctx, live, &drop, retryTestEmbedder{})
	args := map[string]any{
		"q": 1, "r": 0, "placement": "near_anchor", "idempotency_key": "retry-test-write",
		"raw_content": "saved before response loss", "source_id": "test", "confidence": 1,
		"embed": true,
	}
	put := &mcp.CallToolParams{Name: "mosaic_hexxla_put_cell", Arguments: args}
	drop.Store(true)
	if _, err := first.CallTool(ctx, put); err == nil || drop.Load() {
		t.Fatalf("first response was not dropped: %v", err)
	}
	records, err := db.ReadChangelogSince(0, 2)
	if err != nil || len(records) != 1 || records[0].Op != hexxladb.ChangelogOpPutCell {
		t.Fatalf("first call did not commit exactly one put: %v, %v", records, err)
	}
	original, err := db.StatsMVCC()
	if err != nil {
		t.Fatal(err)
	}
	if err := live.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = openMosaicDatabase(path, &hexxladb.Options{})
	if err != nil {
		t.Fatal(err)
	}
	reopened := hexxlastore.NewLiveDB(db)
	t.Cleanup(func() { _ = reopened.Close() })
	retry := connectMutationRetryClient(t, ctx, reopened, new(atomic.Bool), nil)
	res, err := retry.CallTool(ctx, put)
	if err != nil || res.IsError {
		t.Fatalf("retry = %v, %v", res, err)
	}
	var result domain.PutCellMutationResult
	encoded, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatal(err)
	}
	if !result.OK || !result.Embedded || result.Coord != (domain.AxialCoord{Q: 1, R: 0}) {
		t.Fatalf("retry did not return original placement: %+v", result)
	}
	records, err = db.ReadChangelogSince(0, 2)
	if err != nil || len(records) != 1 {
		t.Fatalf("retry duplicated the write: %v, %v", records, err)
	}
	stats, err := db.StatsMVCC()
	if err != nil || stats.CommitSeq != original.CommitSeq {
		t.Fatalf("retry advanced the primary commit sequence: %v, %v", stats.CommitSeq, err)
	}
}

func connectMutationRetryClient(t *testing.T, ctx context.Context, live *hexxlastore.LiveDB, drop *atomic.Bool, embedder secondary.TextEmbedder) *mcp.ClientSession {
	t.Helper()
	gate, err := loadRatchetGate(true, "", false, slog.Default(), nil)
	if err != nil {
		t.Fatal(err)
	}
	server := mcpsrv.NewServer("mutation-retry-test", "test", nil)
	server.AddReceivingMiddleware(gate.Middleware())
	writer := hexxlastore.NewCellWriterAdapter(live, config.DeleteAutoMaintainConfig{})
	if err := mcpsrv.RegisterCoreCellMutationTools(server, services.NewCellMutationService(writer, embedder, 2), nil); err != nil {
		t.Fatal(err)
	}
	tags := services.NewTagCatalogService(hexxlastore.NewTagCatalogAdapter(live))
	if err := mcpsrv.RegisterListTagsTool(server, tags, nil, nil); err != nil {
		t.Fatal(err)
	}
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, dropResponseTransport{sessionTransport{serverTransport}, drop}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "mutation-retry-test", Version: "test"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "mosaic_hexxla_list_tags"})
	if err != nil || res.IsError {
		t.Fatalf("list_tags = %v, %v", res, err)
	}
	return session
}

type retryTestEmbedder struct{}

func (retryTestEmbedder) Embed(context.Context, string, int) ([]float32, error) {
	return []float32{0, 1}, nil
}

type dropResponseTransport struct {
	sessionTransport
	drop *atomic.Bool
}

func (t dropResponseTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	conn, err := t.sessionTransport.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return dropResponseConnection{conn, t.drop}, nil
}

type dropResponseConnection struct {
	mcp.Connection
	drop *atomic.Bool
}

func (c dropResponseConnection) Write(ctx context.Context, msg jsonrpc.Message) error {
	if response, ok := msg.(*jsonrpc.Response); ok && response.Error == nil &&
		!strings.Contains(string(response.Result), `"isError":true`) && c.drop.CompareAndSwap(true, false) {
		_ = c.Close()
		return io.ErrUnexpectedEOF
	}
	return c.Connection.Write(ctx, msg)
}
