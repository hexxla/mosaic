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
	"sync"
	"syscall"
	"time"

	"github.com/hexxla/hexxladb"
	"github.com/hexxla/mcp-ratchet/pkg/ratchet"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/adapter/primary/mcpsrv"
	"github.com/sploitzberg/mosaic/internal/adapter/secondary/hexxlastore"
	"github.com/sploitzberg/mosaic/internal/adapter/secondary/ollamaembed"
	"github.com/sploitzberg/mosaic/internal/config"
	"github.com/sploitzberg/mosaic/internal/core/services"
	ollamac "github.com/sploitzberg/mosaic/internal/ollama"
)

// version is set at link time with -ldflags.
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
	changefeedFlag := flag.Bool("changefeed", false, "enable the HexxlaDB changefeed; an existing default sidecar is continued automatically")
	changefeedPathFlag := flag.String("changefeed-path", "", "use this HexxlaDB changefeed sidecar instead of the default; implies -changefeed")
	reembedFlag := flag.Bool("reembed", false, "regenerate all live-cell embeddings with the configured Ollama model and exit")
	ratchetConfigFlag := flag.String("ratchet-config", "", "path to Ratchet workflow YAML (overrides "+config.EnvRatchetConfigFile+"); a custom path enables Ratchet")
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
	if ratchetObservability.Enabled && !mosaicLoaded.Ratchet && ratchetConfigPath == "" {
		return errors.New("ratchet observability requires ratchet: true, -ratchet-config, or " + config.EnvRatchetConfigFile)
	}
	if ratchetObservability.Enabled && ratchetObservability.Path == mcpCfg.Path {
		return fmt.Errorf("ratchet observability path must differ from MCP path %q", mcpCfg.Path)
	}

	var ratchetObserver *mcpsrv.RatchetObserver
	var ratchetEvents func(ratchet.Event)
	if ratchetObservability.Enabled {
		ratchetObserver = mcpsrv.NewRatchetObserver(ratchetObservability.BearerToken)
		ratchetEvents = ratchetObserver.Observe
	}
	ratchetGate, err := loadRatchetGate(mosaicLoaded.Ratchet, ratchetConfigPath, mosaicLoaded.AdvancedTools, log, ratchetEvents)
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
		"ollama_base", ollamac.DiagnosticURL(ollamaCfg.Base),
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
	openOpts.ChangelogPath = *changefeedPathFlag
	openOpts.ChangelogEnabled = *changefeedFlag || openOpts.ChangelogPath != ""

	// Avoid creating a persistent lock sidecar for an ordinary missing path.
	// OpenExisting still performs the authoritative locked open after this check.
	if _, err := os.Stat(dbCfg.Path); err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	db, err := openMosaicDatabase(dbCfg.Path, openOpts)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	ollamaClient := ollamac.NewClient(ollamaCfg.Base, ollamaCfg.Model)
	embedder := ollamaembed.NewTextEmbedder(ollamaClient)
	if *reembedFlag {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		err := hexxlastore.Reembed(ctx, db, ollamaCfg.Model, embedder)
		if err := errors.Join(err, db.Close()); err != nil {
			return fmt.Errorf("re-embed database: %w", err)
		}
		log.Info("re-embedded database", "path", dbCfg.Path, "model", ollamaCfg.Model)
		return nil
	}
	if err := hexxlastore.EnsureEmbeddingModel(db, ollamaCfg.Model); err != nil {
		return fmt.Errorf("database embedding model: %w", errors.Join(err, db.Close()))
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
	embeddingANN := hexxlastore.NewEmbeddingANNAdapter(live)
	cellRetrieval := services.NewCellRetrievalService(
		cellReader, embeddingANN,
		embedder,
		embeddingDimAtBoot,
	)

	contextPackLoader := hexxlastore.NewContextPackAdapter(live)
	contextAssembly := services.NewContextAssemblyService(contextPackLoader)

	cellWriter := hexxlastore.NewCellWriterAdapter(live, mosaicLoaded.DeleteAutoMaintain)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := cellWriter.FlushPostDeleteMaintain(shutdownCtx); err != nil {
			log.Error("flush post-delete maintain", "err", err)
		}
	}()

	runtimeCfg := config.NewMosaicRuntimeConfig(mosaicLoaded.Retention, mosaicLoaded.AllowDeleteCell)
	mutationSvc := services.NewCellMutationService(
		cellWriter, embedder, embeddingDimAtBoot,
		services.WithMutationPolicy(runtimeCfg.MutationPolicy()),
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
	registrations := []func() error{
		func() error { return mcpsrv.RegisterHealthTool(srv, healthSvc, log, retrievalBudget) },
		func() error { return mcpsrv.RegisterCellSearchTool(srv, cellRetrieval, log, retrievalBudget) },
		func() error { return mcpsrv.RegisterGetCellTool(srv, cellRetrieval, log, retrievalBudget) },
		func() error { return mcpsrv.RegisterContextPackTool(srv, contextAssembly, log, retrievalBudget) },
		func() error { return mcpsrv.RegisterCoreCellMutationTools(srv, mutationSvc, log) },
		func() error { return mcpsrv.RegisterListTagsTool(srv, tagBrowseSvc, log, retrievalBudget) },
	}
	if mosaicLoaded.AdvancedTools {
		registrations = append(registrations,
			func() error { return mcpsrv.RegisterPutEmbeddingTool(srv, mutationSvc, log) },
			func() error { return mcpsrv.RegisterContextBudgetEstimateTool(srv, log) },
			func() error { return mcpsrv.RegisterSeamTools(srv, seamSvc, log, retrievalBudget) },
			func() error { return mcpsrv.RegisterFacetEdgeTools(srv, facetEdgeSvc, log) },
			func() error { return mcpsrv.RegisterFacetEdgeBrowseTools(srv, facetEdgeBrowse, log, retrievalBudget) },
			func() error { return mcpsrv.RegisterTagCountsTool(srv, tagBrowseSvc, log, retrievalBudget) },
			func() error { return mcpsrv.RegisterPersistencePolicyTool(srv, runtimeCfg, configPathUsed, log) },
			func() error { return mcpsrv.RegisterRetrievalBudgetStatusTool(srv, retrievalBudget, log) },
		)
	}
	for _, register := range registrations {
		if err := register(); err != nil {
			return fmt.Errorf("register MCP tools: %w", err)
		}
	}
	activeCalls := &activeToolCalls{drained: make(chan struct{})}
	srv.AddReceivingMiddleware(activeCalls.middleware)
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

	var serveErr error
	serveReturned := false
	select {
	case <-ctx.Done():
	case serveErr = <-errServe:
		serveReturned = true
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	shutdownErr := shutdownMCP(shutdownCtx, httpSrv, srv, activeCalls)
	if !serveReturned {
		serveErr = <-errServe
	}
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	return errors.Join(serveErr, shutdownErr)
}

func openMosaicDatabase(path string, opts *hexxladb.Options) (*hexxladb.DB, error) {
	if opts.ChangelogPath != "" {
		return hexxladb.OpenExisting(path, opts)
	}
	withoutFeed := *opts
	withoutFeed.ChangelogEnabled = false
	db, err := hexxladb.OpenExisting(path, &withoutFeed)
	if err != nil {
		return nil, err
	}

	// Inspect the sidecar while holding the primary lock. A prior writer may
	// have enabled its feed while Mosaic was waiting to open the database.
	_, err = os.Lstat(path + "-changelog")
	if errors.Is(err, os.ErrNotExist) {
		retained, err := hexxlastore.HasRetainedChangefeed(db)
		if err != nil {
			return nil, errors.Join(fmt.Errorf("inspect changefeed history: %w", err), db.Close())
		}
		if retained {
			err := errors.New("database has retained changefeed state but no default sidecar; set -changefeed-path to the matching sidecar or restore the default sidecar")
			return nil, errors.Join(err, db.Close())
		}
		if !opts.ChangelogEnabled {
			return db, nil
		}
	} else if err != nil {
		return nil, errors.Join(fmt.Errorf("inspect default changefeed: %w", err), db.Close())
	}

	if err := db.Close(); err != nil {
		return nil, fmt.Errorf("close before enabling changefeed: %w", err)
	}
	withFeed := *opts
	withFeed.ChangelogEnabled = true
	return hexxladb.OpenExisting(path, &withFeed)
}

// shutdownMCP stops accepting HTTP work and drains active tool calls before
// closing idle MCP sessions. Closing sessions first would stop in-flight calls
// from receiving their normal responses.
func shutdownMCP(ctx context.Context, httpSrv *http.Server, srv *mcp.Server, calls *activeToolCalls) error {
	drained := calls.stop()
	shutdownErr := httpSrv.Shutdown(ctx)
	select {
	case <-drained:
	case <-ctx.Done():
		shutdownErr = errors.Join(shutdownErr, ctx.Err())
	}
	if shutdownErr != nil {
		shutdownErr = errors.Join(shutdownErr, httpSrv.Close())
	}
	var closeErr error
	for session := range srv.Sessions() {
		closeErr = errors.Join(closeErr, session.Close())
	}
	return errors.Join(shutdownErr, closeErr)
}

func loadRatchetGate(enabled bool, path string, advanced bool, log *slog.Logger, observer func(ratchet.Event)) (*mcpsrv.RatchetGate, error) {
	if !enabled && path == "" {
		return nil, nil
	}
	policy, err := config.LoadRatchetPolicy(path)
	if err != nil {
		return nil, err
	}
	if err := mcpsrv.NewRatchetGate(nil, policy).ValidateToolNames(); err != nil {
		return nil, err
	}
	policy, err = mcpsrv.PolicyForTools(policy, advanced)
	if err != nil {
		return nil, err
	}
	engine, err := ratchet.New(policy, ratchet.Options{Observer: observer})
	if err != nil {
		return nil, err
	}
	gate := mcpsrv.NewRatchetGate(engine, policy)
	if err := gate.ValidateProtectedTools(mcpsrv.MutationToolNames(advanced)); err != nil {
		return nil, err
	}
	log.Info("ratchet enforcement enabled", "custom_policy", path != "", "rules", len(policy.Rules))
	return gate, nil
}

// activeToolCalls closes admission during shutdown and tracks handlers even when
// their HTTP connection has already gone away.
type activeToolCalls struct {
	mu       sync.Mutex
	active   int
	stopping bool
	drained  chan struct{}
}

func (c *activeToolCalls) middleware(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		if method != "tools/call" {
			return next(ctx, method, req)
		}
		c.mu.Lock()
		if c.stopping {
			c.mu.Unlock()
			result := &mcp.CallToolResult{}
			result.SetError(errors.New("mosaic MCP server is shutting down"))
			return result, nil
		}
		c.active++
		c.mu.Unlock()
		defer func() {
			c.mu.Lock()
			defer c.mu.Unlock()
			c.active--
			if c.stopping && c.active == 0 {
				close(c.drained)
			}
		}()
		return next(ctx, method, req)
	}
}

func (c *activeToolCalls) stop() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.stopping {
		c.stopping = true
		if c.active == 0 {
			close(c.drained)
		}
	}
	return c.drained
}
