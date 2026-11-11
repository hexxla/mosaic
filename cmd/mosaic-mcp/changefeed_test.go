package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hexxla/hexxladb"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hexxla/mosaic/internal/adapter/primary/mcpsrv"
	"github.com/hexxla/mosaic/internal/adapter/secondary/hexxlastore"
	"github.com/hexxla/mosaic/internal/config"
	"github.com/hexxla/mosaic/internal/core/services"
)

func TestOpenMosaicDatabaseWithCustomChangefeed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.hexxla")
	sidecar := filepath.Join(t.TempDir(), "custom.log")
	opts := &hexxladb.Options{ChangelogEnabled: true, ChangelogPath: sidecar}
	db, err := hexxladb.Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.AdvanceChangelogConsumer("replica", 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	withoutPath, err := openMosaicDatabase(path, &hexxladb.Options{})
	if withoutPath != nil {
		_ = withoutPath.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "-changefeed-path") {
		t.Fatalf("open without custom sidecar = %v, want guidance for registered consumers", err)
	}

	reopened, err := openMosaicDatabase(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	if _, found, err := reopened.GetChangelogConsumerCursor("replica"); err != nil || !found {
		t.Fatalf("reopened consumer found = %t, err = %v", found, err)
	}
	if _, err := reopened.ReadChangelogSince(0, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + "-changelog"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("default sidecar stat = %v, want no default sidecar", err)
	}
}

func TestOpenMosaicDatabaseWithoutChangefeed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.hexxla")
	db, err := hexxladb.Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := openMosaicDatabase(path, &hexxladb.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	if _, err := reopened.ReadChangelogSince(0, 1); !errors.Is(err, hexxladb.ErrChangelogDisabled) {
		t.Fatalf("read feed = %v, want disabled", err)
	}
}

func TestOpenMosaicDatabaseRejectsOmittedCustomChangefeedWithoutConsumers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.hexxla")
	opts := &hexxladb.Options{ChangelogEnabled: true, ChangelogPath: filepath.Join(t.TempDir(), "custom.log")}
	db, err := hexxladb.Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	key, err := hexxladb.Pack(hexxladb.Coord{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Update(func(tx *hexxladb.Tx) error {
		return tx.PutCell(t.Context(), hexxladb.NewFactCell(key, "retained history", "test", "test", 1))
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	for _, enableDefault := range []bool{false, true} {
		opened, err := openMosaicDatabase(path, &hexxladb.Options{ChangelogEnabled: enableDefault})
		if opened != nil {
			_ = opened.Close()
		}
		if err == nil || !strings.Contains(err.Error(), "-changefeed-path") {
			t.Fatalf("open without custom path (enable default=%t) = %v, want path guidance", enableDefault, err)
		}
	}
	if _, err := os.Stat(path + "-changelog"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("default sidecar stat = %v, want no default sidecar", err)
	}
	reopened, err := openMosaicDatabase(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	if records, err := reopened.ReadChangelogSince(0, 2); err != nil || len(records) != 1 {
		t.Fatalf("custom feed after reopen = %v, %v", records, err)
	}
}

func TestRatchetGovernedWriteContinuesChangefeed(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	path := filepath.Join(t.TempDir(), "memory.hexxla")
	original, err := hexxladb.Open(path, &hexxladb.Options{ChangelogEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = original.Close() })
	key, err := hexxladb.Pack(hexxladb.Coord{})
	if err != nil {
		t.Fatal(err)
	}
	if err := original.Update(func(tx *hexxladb.Tx) error {
		return tx.PutCell(ctx, hexxladb.NewFactCell(key, "existing memory", "fixture", "test", 1))
	}); err != nil {
		t.Fatal(err)
	}
	history, err := original.ReadChangelogSince(0, 2)
	if err != nil || len(history) != 1 {
		t.Fatalf("existing feed = %v, %v", history, err)
	}
	if err := original.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := openMosaicDatabase(path, &hexxladb.Options{})
	if err != nil {
		t.Fatal(err)
	}
	live := hexxlastore.NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })
	gate, err := loadRatchetGate(true, "", false, slog.Default(), nil)
	if err != nil {
		t.Fatal(err)
	}
	server := mcpsrv.NewServer("mosaic-changefeed-test", "test", nil)
	server.AddReceivingMiddleware(gate.Middleware())
	writer := hexxlastore.NewCellWriterAdapter(live, config.DeleteAutoMaintainConfig{})
	if err := mcpsrv.RegisterCoreCellMutationTools(server, services.NewCellMutationService(writer, nil, 0), nil); err != nil {
		t.Fatal(err)
	}
	tags := services.NewTagCatalogService(hexxlastore.NewTagCatalogAdapter(live))
	if err := mcpsrv.RegisterListTagsTool(server, tags, nil, nil); err != nil {
		t.Fatal(err)
	}

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, sessionTransport{serverTransport}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "changefeed-test", Version: "test"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })

	put := &mcp.CallToolParams{
		Name: "mosaic_hexxla_put_cell",
		Arguments: map[string]any{
			"q": 1, "r": 0, "raw_content": "Ratchet-governed memory", "source_id": "integration-test", "confidence": 1,
		},
	}
	result, err := session.CallTool(ctx, put)
	if err != nil || !result.IsError || len(result.Content) == 0 {
		t.Fatalf("put without list_tags = %v, %v", result, err)
	}
	message, ok := result.Content[0].(*mcp.TextContent)
	if !ok || !strings.Contains(message.Text, "list_tags") {
		t.Fatalf("expected Ratchet denial, got %v", result.Content)
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "mosaic_hexxla_list_tags"})
	if err != nil || result.IsError {
		t.Fatalf("list_tags = %v, %v", result, err)
	}
	result, err = session.CallTool(ctx, put)
	if err != nil || result.IsError {
		t.Fatalf("governed put = %v, %v", result, err)
	}
	records, err := db.ReadChangelogSince(history[0].Seq, 2)
	if err != nil || len(records) != 1 || records[0].Op != hexxladb.ChangelogOpPutCell || records[0].Seq != history[0].Seq+1 {
		t.Fatalf("new feed entries = %v, %v; want one put after sequence %d", records, err, history[0].Seq)
	}
	if !bytes.Contains(records[0].Inline, []byte("Ratchet-governed memory")) {
		t.Fatalf("feed entry does not contain the MCP cell: %v", records[0])
	}
}

// The SDK's in-memory transport uses an empty session ID by default.
type sessionTransport struct{ *mcp.InMemoryTransport }

func (t sessionTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	conn, err := t.InMemoryTransport.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return identifiedConnection{conn}, nil
}

type identifiedConnection struct{ mcp.Connection }

func (identifiedConnection) SessionID() string { return "changefeed-test-session" }
