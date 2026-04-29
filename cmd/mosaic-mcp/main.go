// Command mosaic-mcp runs the local MCP Streamable HTTP server backed by HexxlaDB tools.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/go-llm-project-structure/internal/adapter/primary/mcpsrv"
	"github.com/sploitzberg/go-llm-project-structure/internal/adapter/secondary/hexxlastore"
	"github.com/sploitzberg/go-llm-project-structure/internal/adapter/secondary/ollamaembed"
	"github.com/sploitzberg/go-llm-project-structure/internal/config"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/services"
	ollamac "github.com/sploitzberg/go-llm-project-structure/internal/ollama"
)

// version is set at link time by goreleaser or -ldflags.
var version = "dev"

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := run(log); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	policyFlag := flag.String("policy", "", "path to Mosaic config YAML (retention section; overrides MOSAIC_POLICY_FILE)")
	flag.Parse()

	mcpCfg, err := config.LoadMCPFromEnv()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	dbCfg, err := config.LoadDBFromEnv()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	ollamaCfg, err := config.LoadOllamaFromEnv()
	if err != nil {
		return fmt.Errorf("config ollama: %w", err)
	}

	configPath := config.ResolveMosaicConfigPath(*policyFlag)
	mosaicLoaded := config.DefaultMosaicConfig()
	configPathUsed := configPath
	if configPath != "" {
		loaded, err := config.LoadMosaicConfigFromFile(configPath)
		if err != nil {
			return fmt.Errorf("mosaic config: %w", err)
		}
		mosaicLoaded = loaded
	} else {
		configPathUsed = ""
	}
	log.Info("mosaic config",
		"capture_mode", mosaicLoaded.Retention.CaptureMode,
		"enforcement", mosaicLoaded.Retention.Enforcement,
		"allow_delete_cell", mosaicLoaded.AllowDeleteCell,
		"config_file", configPathUsed)

	db, err := hexxladb.Open(dbCfg.Path, nil)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			log.Error("close database", "err", cerr)
		}
	}()

	engineHealth := hexxlastore.NewEngineHealthAdapter(db)
	healthSvc := services.NewHealthService(engineHealth, version)

	cellReader := hexxlastore.NewCellReaderAdapter(db)
	ollamaClient := ollamac.NewClient(ollamaCfg.Base, ollamaCfg.Model)
	cellRetrieval := services.NewCellRetrievalService(
		cellReader,
		ollamaembed.NewTextEmbedder(ollamaClient),
		db.EmbeddingDimension(),
	)

	contextPackLoader := hexxlastore.NewContextPackAdapter(db)
	contextAssembly := services.NewContextAssemblyService(contextPackLoader)

	embeddingANN := hexxlastore.NewEmbeddingANNAdapter(db, ollamaClient)
	embeddingSvc := services.NewEmbeddingSearchService(embeddingANN)

	cellWriter := hexxlastore.NewCellWriterAdapter(db)
	runtimeCfg := config.NewMosaicRuntimeConfig(mosaicLoaded.Retention, mosaicLoaded.AllowDeleteCell)
	mutationSvc := services.NewCellMutationService(
		cellWriter, ollamaembed.NewTextEmbedder(ollamaClient), db.EmbeddingDimension(),
		services.WithMosaicRuntime(runtimeCfg),
	)

	seamStore := hexxlastore.NewSeamStoreAdapter(db)
	seamSvc := services.NewSeamLifecycleService(seamStore)

	facetEdgeStore := hexxlastore.NewFacetEdgeStoreAdapter(db)
	facetEdgeSvc := services.NewFacetEdgeService(facetEdgeStore)
	facetEdgeBrowse := services.NewFacetEdgeReadService(facetEdgeStore)

	tagCatalog := hexxlastore.NewTagCatalogAdapter(db)
	tagBrowseSvc := services.NewTagCatalogService(tagCatalog)

	policyInstructions := config.MCPPolicyInstructions(runtimeCfg, configPathUsed)
	srv := mcpsrv.NewServer("mosaic", version, mcpsrv.ServerInstructions(policyInstructions))
	mcpsrv.RegisterHealthTool(srv, healthSvc, log)
	mcpsrv.RegisterCellQueryTool(srv, cellRetrieval, log)
	mcpsrv.RegisterCellSearchTool(srv, cellRetrieval, log)
	mcpsrv.RegisterEmbeddingSearchTool(srv, embeddingSvc, log)
	mcpsrv.RegisterContextPackTool(srv, contextAssembly, log)
	mcpsrv.RegisterContextBudgetEstimateTool(srv, log)
	mcpsrv.RegisterCellMutationTools(srv, mutationSvc, runtimeCfg, log)
	mcpsrv.RegisterSeamTools(srv, seamSvc, log)
	mcpsrv.RegisterFacetEdgeTools(srv, facetEdgeSvc, log)
	mcpsrv.RegisterFacetEdgeBrowseTools(srv, facetEdgeBrowse, log)
	mcpsrv.RegisterTagBrowseTools(srv, tagBrowseSvc, log)
	mcpsrv.RegisterPersistencePolicyTool(srv, runtimeCfg, configPathUsed, log)
	h := mcpsrv.StreamableHTTPHandler(srv, log)

	mux := http.NewServeMux()
	for _, method := range [...]string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		mux.Handle(method+" "+mcpCfg.Path, h)
	}

	httpSrv := &http.Server{
		Addr:              mcpCfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errServe := make(chan error, 1)
	go func() {
		log.Info("mosaic MCP listening", "addr", mcpCfg.Addr, "path", mcpCfg.Path, "db", dbCfg.Path, "version", version)
		errServe <- httpSrv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http: %w", err)
		}
		firstErr := <-errServe
		if firstErr != nil && !errors.Is(firstErr, http.ErrServerClosed) {
			return firstErr
		}
	case err := <-errServe:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http serve: %w", err)
		}
	}

	return nil
}
