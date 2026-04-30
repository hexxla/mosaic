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

	"github.com/sploitzberg/mosaic/internal/adapter/primary/mcpsrv"
	"github.com/sploitzberg/mosaic/internal/adapter/secondary/hexxlastore"
	"github.com/sploitzberg/mosaic/internal/adapter/secondary/ollamaembed"
	"github.com/sploitzberg/mosaic/internal/config"
	"github.com/sploitzberg/mosaic/internal/core/services"
	ollamac "github.com/sploitzberg/mosaic/internal/ollama"
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
	policyFlag := flag.String("policy", "", "path to Mosaic config YAML (retention section; overrides "+config.EnvPolicyFile+")")
	dbPassphraseFlag := flag.String("db-passphrase", "", "optional HexxlaDB encryption passphrase (overrides "+config.EnvDBPassphrase+"; avoid on shared hosts — visible in process list)")
	dbFlag := flag.String("db", "", "path to HexxlaDB file (overrides -name and "+config.EnvDBPath+")")
	nameFlag := flag.String("name", "", "base name: <db-dir>/<name>.hexxla (overrides "+config.EnvDBPath+"; mutually exclusive with -db)")
	dbDirFlag := flag.String("db-dir", "", "parent directory when using -name (default: from "+config.EnvMosaicDBDir+" or .tmp)")
	flag.Parse()

	mcpCfg, err := config.LoadMCPFromEnv()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	dbPath, err := config.ResolveMosaicDBPath(config.MosaicDBPathInput{
		DBFlag:    *dbFlag,
		NameFlag:  *nameFlag,
		DBDirFlag: *dbDirFlag,
	}, false)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	dbCfg := config.DB{Path: dbPath}

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

	ollamaCfg, err := config.ResolveOllama(config.OllamaResolveInput{
		YAMLBaseURL:    mosaicLoaded.OllamaBaseURL,
		YAMLEmbedModel: mosaicLoaded.OllamaEmbedModel,
	})
	if err != nil {
		return fmt.Errorf("config ollama: %w", err)
	}

	log.Info("mosaic config",
		"capture_mode", mosaicLoaded.Retention.CaptureMode,
		"enforcement_enabled", mosaicLoaded.Retention.EnforcementEnabled(),
		"allow_delete_cell", mosaicLoaded.AllowDeleteCell,
		"retrieval_session_approx_token_budget", mosaicLoaded.Retrieval.SessionApproxTokenBudget,
		"auto_maintain_after_cell_delete", mosaicLoaded.DeleteAutoMaintain.Enabled,
		"post_delete_maintain_debounce_ms", mosaicLoaded.DeleteAutoMaintain.DebounceAfterDelete.Milliseconds(),
		"mvcc_retain_commits_behind_head", mosaicLoaded.MVCCRetainCommitsBehindHead,
		"ollama_base", ollamaCfg.Base.String(),
		"ollama_embed_model", ollamaCfg.Model,
		"config_file", configPathUsed)

	retrievalBudget := mcpsrv.NewRetrievalBudgetTracker(mosaicLoaded.Retrieval)

	encOpts, err := config.BuildHexxlaOpenOptions(config.HexxlaOpenParams{
		FlagPassphrase: *dbPassphraseFlag,
		YAMLPassphrase: mosaicLoaded.DatabasePassphrase,
	})
	if err != nil {
		return fmt.Errorf("database encryption options: %w", err)
	}

	openOpts := config.MergeMVCCRetainIntoOpenOptions(encOpts, mosaicLoaded.MVCCRetainCommitsBehindHead)

	db, err := hexxladb.Open(dbCfg.Path, openOpts)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	embeddingDimAtBoot := db.EmbeddingDimension()

	live := hexxlastore.NewLiveDB(db)
	defer func() {
		if cerr := live.Close(); cerr != nil {
			log.Error("close database", "err", cerr)
		}
	}()

	engineHealth := hexxlastore.NewEngineHealthAdapter(live, dbCfg.Path)
	healthSvc := services.NewHealthService(engineHealth, version, mosaicLoaded.MVCCRetainCommitsBehindHead)

	cellReader := hexxlastore.NewCellReaderAdapter(live)
	ollamaClient := ollamac.NewClient(ollamaCfg.Base, ollamaCfg.Model)
	cellRetrieval := services.NewCellRetrievalService(
		cellReader,
		ollamaembed.NewTextEmbedder(ollamaClient),
		embeddingDimAtBoot,
	)

	contextPackLoader := hexxlastore.NewContextPackAdapter(live)
	contextAssembly := services.NewContextAssemblyService(contextPackLoader)

	embeddingANN := hexxlastore.NewEmbeddingANNAdapter(live, ollamaClient)
	embeddingSvc := services.NewEmbeddingSearchService(embeddingANN)

	cellWriter := hexxlastore.NewCellWriterAdapter(live, dbCfg.Path, openOpts, mosaicLoaded.DeleteAutoMaintain)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := cellWriter.FlushPostDeleteMaintain(shutdownCtx); err != nil {
			log.Error("flush post-delete maintain", "err", err)
		}
	}()

	runtimeCfg := config.NewMosaicRuntimeConfig(mosaicLoaded.Retention, mosaicLoaded.AllowDeleteCell)
	mutationSvc := services.NewCellMutationService(
		cellWriter, ollamaembed.NewTextEmbedder(ollamaClient), embeddingDimAtBoot,
		services.WithMosaicRuntime(runtimeCfg),
	)

	seamStore := hexxlastore.NewSeamStoreAdapter(live)
	seamSvc := services.NewSeamLifecycleService(seamStore)

	facetEdgeStore := hexxlastore.NewFacetEdgeStoreAdapter(live)
	facetEdgeSvc := services.NewFacetEdgeService(facetEdgeStore)
	facetEdgeBrowse := services.NewFacetEdgeReadService(facetEdgeStore)

	tagCatalog := hexxlastore.NewTagCatalogAdapter(live)
	tagBrowseSvc := services.NewTagCatalogService(tagCatalog)

	policyInstructions := config.MCPPolicyInstructions(runtimeCfg, configPathUsed)
	srv := mcpsrv.NewServer("mosaic", version, mcpsrv.ServerInstructions(policyInstructions))
	mcpsrv.RegisterHealthTool(srv, healthSvc, log, retrievalBudget)
	mcpsrv.RegisterCellQueryTool(srv, cellRetrieval, log, retrievalBudget)
	mcpsrv.RegisterCellSearchTool(srv, cellRetrieval, log, retrievalBudget)
	mcpsrv.RegisterEmbeddingSearchTool(srv, embeddingSvc, log, retrievalBudget)
	mcpsrv.RegisterContextPackTool(srv, contextAssembly, log, retrievalBudget)
	mcpsrv.RegisterContextBudgetEstimateTool(srv, log)
	mcpsrv.RegisterCellMutationTools(srv, mutationSvc, runtimeCfg, log)
	mcpsrv.RegisterSeamTools(srv, seamSvc, log, retrievalBudget)
	mcpsrv.RegisterFacetEdgeTools(srv, facetEdgeSvc, log)
	mcpsrv.RegisterFacetEdgeBrowseTools(srv, facetEdgeBrowse, log, retrievalBudget)
	mcpsrv.RegisterTagBrowseTools(srv, tagBrowseSvc, log, retrievalBudget)
	mcpsrv.RegisterPersistencePolicyTool(srv, runtimeCfg, configPathUsed, log)
	mcpsrv.RegisterRetrievalBudgetStatusTool(srv, retrievalBudget, log)
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
