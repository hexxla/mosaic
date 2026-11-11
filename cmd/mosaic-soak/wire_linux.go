//go:build linux

package main

import (
	"context"
	"fmt"
	"net/http/httptest"
	"time"

	"github.com/hexxla/mcp-ratchet/pkg/ratchet"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hexxla/mosaic/internal/adapter/primary/mcpsrv"
	"github.com/hexxla/mosaic/internal/adapter/secondary/hexxlastore"
	"github.com/hexxla/mosaic/internal/config"
	"github.com/hexxla/mosaic/internal/core/ports/secondary"
	"github.com/hexxla/mosaic/internal/core/services"
)

type wire struct {
	ctx     context.Context
	server  *mcp.Server
	http    *httptest.Server
	session *mcp.ClientSession
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func require(ok bool, message string) {
	if !ok {
		panic(message)
	}
}
func openWire(ctx context.Context, live *hexxlastore.LiveDB, path string, advanced bool, dim uint16, model string, embedder secondary.TextEmbedder) *wire {
	policy, err := config.LoadRatchetPolicy("")
	must(err)
	policy, err = mcpsrv.PolicyForTools(policy, advanced)
	must(err)
	engine, err := ratchet.New(policy, ratchet.Options{})
	must(err)
	gate := mcpsrv.NewRatchetGate(engine, policy)
	must(gate.ValidateToolNames())
	must(gate.ValidateProtectedTools(mcpsrv.MutationToolNames(advanced)))
	server := mcpsrv.NewServer("qualification", "alpha-baseline", nil)
	server.AddReceivingMiddleware(gate.Middleware())
	reader := hexxlastore.NewCellReaderAdapter(live)
	cells := services.NewCellRetrievalService(reader, hexxlastore.NewEmbeddingANNAdapter(live), embedder, dim, model)
	runtime := config.NewMosaicRuntimeConfig(config.DefaultRetentionPolicy(), true)
	writer := hexxlastore.NewCellWriterAdapter(live, config.DeleteAutoMaintainConfig{})
	mutations := services.NewCellMutationService(writer, embedder, dim, services.WithMutationPolicy(runtime.MutationPolicy()), services.WithEmbeddingModel(model))
	tags := services.NewTagCatalogService(hexxlastore.NewTagCatalogAdapter(live))
	contextService := services.NewContextAssemblyService(hexxlastore.NewContextPackAdapter(live))
	health := services.NewHealthService(hexxlastore.NewEngineHealthAdapter(live, path), "qualification", 0)
	regs := []func() error{
		func() error { return mcpsrv.RegisterHealthTool(server, health, nil, nil) },
		func() error { return mcpsrv.RegisterCellSearchTool(server, cells, nil, nil) },
		func() error { return mcpsrv.RegisterGetCellTool(server, cells, nil, nil) },
		func() error { return mcpsrv.RegisterContextPackTool(server, contextService, nil, nil) },
		func() error { return mcpsrv.RegisterCoreCellMutationTools(server, mutations, nil) },
		func() error { return mcpsrv.RegisterListTagsTool(server, tags, nil, nil) },
	}
	if advanced {
		explorerStore := hexxlastore.NewMemoryExplorerAdapter(live)
		explorer := services.NewMemoryExplorationService(explorerStore)
		seamStore := hexxlastore.NewSeamStoreAdapter(live)
		seams := services.NewSeamLifecycleService(seamStore)
		facetStore := hexxlastore.NewFacetEdgeStoreAdapter(live)
		regs = append(regs,
			func() error {
				return mcpsrv.RegisterBrowseTools(server, services.NewMemoryBrowseService(explorerStore), nil)
			},
			func() error {
				return mcpsrv.RegisterExplorationTools(server, explorer, services.NewEmbeddingMaintenanceService(explorerStore, true), nil)
			},
			func() error { return mcpsrv.RegisterCellBatchTool(server, mutations) },
			func() error { return mcpsrv.RegisterPutEmbeddingTool(server, mutations, nil) },
			func() error { return mcpsrv.RegisterContextBudgetEstimateTool(server, nil) },
			func() error { return mcpsrv.RegisterSeamRecordTool(server, services.NewSeamRecordService(seamStore)) },
			func() error { return mcpsrv.RegisterSeamTools(server, seams, nil, nil) },
			func() error {
				return mcpsrv.RegisterFacetEdgeTools(server, services.NewFacetEdgeService(facetStore), nil)
			},
			func() error {
				return mcpsrv.RegisterFacetEdgeBrowseTools(server, services.NewFacetEdgeReadService(facetStore), nil, nil)
			},
			func() error { return mcpsrv.RegisterTagCountsTool(server, tags, nil, nil) },
			func() error {
				return mcpsrv.RegisterPersistencePolicyTool(server, runtime, "synthetic qualification", nil)
			},
			func() error { return mcpsrv.RegisterRetrievalBudgetStatusTool(server, nil, nil) },
		)
	}
	for _, reg := range regs {
		must(reg())
	}
	httpServer := httptest.NewServer(mcpsrv.StreamableHTTPHandler(server, nil))
	client := mcp.NewClient(&mcp.Implementation{Name: "qualification", Version: "alpha"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	must(err)
	tools, err := session.ListTools(ctx, nil)
	must(err)
	expected := 7
	if advanced {
		expected = 43
	}
	require(len(tools.Tools) == expected, fmt.Sprintf("tool inventory=%d, want %d", len(tools.Tools), expected))
	return &wire{ctx: ctx, server: server, http: httpServer, session: session}
}
func (w *wire) close() {
	must(w.session.Close())
	for session := range w.server.Sessions() {
		must(session.Close())
	}
	w.http.Close()
}
func (w *wire) denied(name string, args any) {
	ctx, cancel := context.WithTimeout(w.ctx, 30*time.Second)
	defer cancel()
	result, err := w.session.CallTool(ctx, &mcp.CallToolParams{Name: "mosaic_hexxla_" + name, Arguments: args})
	must(err)
	require(result.IsError, "Ratchet admitted write without fresh prerequisite")
}
func (w *wire) call(name string, args any) *mcp.CallToolResult {
	ctx, cancel := context.WithTimeout(w.ctx, 30*time.Second)
	defer cancel()
	result, err := w.session.CallTool(ctx, &mcp.CallToolParams{Name: "mosaic_hexxla_" + name, Arguments: args})
	must(err)
	if result.IsError {
		var msg string
		for _, c := range result.Content {
			if text, ok := c.(*mcp.TextContent); ok {
				msg += text.Text
			}
		}
		panic(name + ": " + msg)
	}
	return result
}
