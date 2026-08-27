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
	ratchetadapters "github.com/hexxla/mcp-ratchet/pkg/ratchet/adapters"
	ratchetsecondary "github.com/hexxla/mcp-ratchet/pkg/ratchet/ports/secondary"
	ratchetservices "github.com/hexxla/mcp-ratchet/pkg/ratchet/services"

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
	ratchetConfigFlag := flag.String("ratchet-config", "", "path to Ratchet workflow YAML (overrides "+config.EnvRatchetConfigFile+"); omitted disables Ratchet")
	ratchetObservabilityTokenFileFlag := flag.String("ratchet-observability-token-file", "", "private bearer-token file; enables the authenticated live Ratchet WebSocket stream")
	ratchetObservabilityPathFlag := flag.String("ratchet-observability-path", config.DefaultRatchetObservabilityPath, "live Ratchet WebSocket endpoint path")
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

	ratchetConfigPath := config.ResolveRatchetConfigPath(*ratchetConfigFlag)
	ratchetObservability, err := config.LoadRatchetObservability(*ratchetObservabilityTokenFileFlag, *ratchetObservabilityPathFlag)
	if err != nil {
		return fmt.Errorf("ratchet observability setup: %w", err)
	}
	if ratchetObservability.Enabled && ratchetConfigPath == "" {
		return errors.New("ratchet observability requires -ratchet-config or " + config.EnvRatchetConfigFile)
	}
	if ratchetObservability.Enabled && ratchetObservability.Path == mcpCfg.Path {
		return fmt.Errorf("ratchet observability path must differ from MCP path %q", mcpCfg.Path)
	}

	var ratchetObserver *mcpsrv.RatchetObserver
	var ratchetEventStore ratchetsecondary.EventStore
	if ratchetObservability.Enabled {
		ratchetObserver = mcpsrv.NewRatchetObserver(ratchetObservability.BearerToken)
		ratchetEventStore = ratchetObserver
	}
	ratchetGate, err := loadRatchetGate(ratchetConfigPath, log, ratchetEventStore)
	if err != nil {
		return fmt.Errorf("ratchet setup: %w", err)
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
	if ratchetGate != nil {
		srv.AddReceivingMiddleware(ratchetGate.Middleware())
	}
	registrations := [...]func() error{
		func() error { return mcpsrv.RegisterHealthTool(srv, healthSvc, log, retrievalBudget) },
		func() error { return mcpsrv.RegisterCellQueryTool(srv, cellRetrieval, log, retrievalBudget) },
		func() error { return mcpsrv.RegisterCellSearchTool(srv, cellRetrieval, log, retrievalBudget) },
		func() error { return mcpsrv.RegisterEmbeddingSearchTool(srv, embeddingSvc, log, retrievalBudget) },
		func() error { return mcpsrv.RegisterContextPackTool(srv, contextAssembly, log, retrievalBudget) },
		func() error { return mcpsrv.RegisterContextBudgetEstimateTool(srv, log) },
		func() error { return mcpsrv.RegisterCellMutationTools(srv, mutationSvc, runtimeCfg, log) },
		func() error { return mcpsrv.RegisterSeamTools(srv, seamSvc, log, retrievalBudget) },
		func() error { return mcpsrv.RegisterFacetEdgeTools(srv, facetEdgeSvc, log) },
		func() error { return mcpsrv.RegisterFacetEdgeBrowseTools(srv, facetEdgeBrowse, log, retrievalBudget) },
		func() error { return mcpsrv.RegisterTagBrowseTools(srv, tagBrowseSvc, log, retrievalBudget) },
		func() error { return mcpsrv.RegisterPersistencePolicyTool(srv, runtimeCfg, configPathUsed, log) },
		func() error { return mcpsrv.RegisterRetrievalBudgetStatusTool(srv, retrievalBudget, log) },
	}
	for _, register := range registrations {
		if err := register(); err != nil {
			return fmt.Errorf("register MCP tools: %w", err)
		}
	}
	h := mcpsrv.StreamableHTTPHandler(srv, log)

	mux := http.NewServeMux()
	for _, method := range [...]string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		mux.Handle(method+" "+mcpCfg.Path, h)
	}
	if ratchetObserver != nil {
		mux.Handle("GET "+ratchetObservability.Path, ratchetObserver)
		log.Info("Ratchet live observability enabled", "path", ratchetObservability.Path)
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

func loadRatchetGate(path string, log *slog.Logger, eventStore ratchetsecondary.EventStore) (*mcpsrv.RatchetGate, error) {
	loaded, err := config.LoadRatchetConfig(path)
	if err != nil {
		return nil, fmt.Errorf("ratchet config: %w", err)
	}
	if loaded.Path == "" {
		return nil, nil
	}

	// #nosec G304 -- the operator explicitly selects this configuration path.
	configFile, err := os.Open(loaded.Path)
	if err != nil {
		return nil, fmt.Errorf("open ratchet config: %w", err)
	}

	sessions := ratchetadapters.NewMemorySessionStore()
	configLoader := ratchetadapters.NewYAMLConfigLoader()
	tokenStore := ratchetadapters.NewMemoryTokenStore()
	randomGenerator := ratchetadapters.NewCryptoRandomGenerator()
	clock := ratchetadapters.NewRealClock()
	service := ratchetservices.NewRatchetService(configLoader, tokenStore, sessions, randomGenerator, clock)
	if eventStore != nil {
		service = ratchetservices.NewRatchetServiceWithObservability(configLoader, tokenStore, sessions, randomGenerator, clock, eventStore)
	}
	rules, loadErr := service.LoadConfiguration(context.Background(), configFile)
	closeErr := configFile.Close()
	if loadErr != nil {
		return nil, fmt.Errorf("load ratchet config: %w", loadErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close ratchet config: %w", closeErr)
	}

	gate := mcpsrv.NewRatchetGate(service, sessions, rules, log)
	if err := gate.ValidateProtectedTools(mcpsrv.MutationToolNames()); err != nil {
		return nil, err
	}
	log.Info("ratchet enforcement enabled", "config_file", loaded.Path, "rules", len(rules))
	return gate, nil
}
