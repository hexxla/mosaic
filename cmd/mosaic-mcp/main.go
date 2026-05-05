// Command mosaic-mcp runs the local MCP Streamable HTTP server backed by HexxlaDB tools.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hexxla/hexxladb"
	ratchetadapters "github.com/hexxla/mcp-ratchet/pkg/ratchet/adapters"
	ratchetdomain "github.com/hexxla/mcp-ratchet/pkg/ratchet/domain"
	ratchetprimary "github.com/hexxla/mcp-ratchet/pkg/ratchet/ports/primary"
	ratchetsecondary "github.com/hexxla/mcp-ratchet/pkg/ratchet/ports/secondary"
	ratchetservices "github.com/hexxla/mcp-ratchet/pkg/ratchet/services"

	"github.com/sploitzberg/mosaic/internal/adapter/primary/mcpsrv"
	"github.com/sploitzberg/mosaic/internal/adapter/secondary/hexxlastore"
	"github.com/sploitzberg/mosaic/internal/adapter/secondary/ollamaembed"
	"github.com/sploitzberg/mosaic/internal/config"
	mosaicservices "github.com/sploitzberg/mosaic/internal/core/services"
	ollamac "github.com/sploitzberg/mosaic/internal/ollama"
)

// version is set at link time by goreleaser or -ldflags.
var version = "dev"

// WebSocket upgrader configuration
var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for localhost development
	},
}

// eventBroadcaster manages WebSocket connections and broadcasts events
type eventBroadcaster struct {
	mu          sync.RWMutex
	connections map[ratchetdomain.SessionID]map[*websocket.Conn]struct{}
}

func newEventBroadcaster() *eventBroadcaster {
	return &eventBroadcaster{
		connections: make(map[ratchetdomain.SessionID]map[*websocket.Conn]struct{}),
	}
}

func (b *eventBroadcaster) subscribe(sessionID ratchetdomain.SessionID, conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.connections[sessionID] == nil {
		b.connections[sessionID] = make(map[*websocket.Conn]struct{})
	}
	b.connections[sessionID][conn] = struct{}{}
}

func (b *eventBroadcaster) unsubscribe(sessionID ratchetdomain.SessionID, conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if conns, ok := b.connections[sessionID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(b.connections, sessionID)
		}
	}
}

// Broadcast implements EventBroadcaster interface.
// Sends the event to all WebSocket clients subscribed to this session.
func (b *eventBroadcaster) Broadcast(sessionID ratchetdomain.SessionID, event *ratchetdomain.Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	conns := b.connections[sessionID]
	for conn := range conns {
		_ = conn.WriteJSON(event) // Best-effort broadcast
	}
}

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
	ratchetConfigFlag := flag.String("ratchet-config", "", "path to ratchet workflow enforcement YAML (overrides "+config.EnvRatchetConfigFile+"); if not provided, ratchet is disabled")
	flag.Parse()

	mcpCfg, err := config.LoadMCPFromEnv()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// Merge MCP observability from mosaic config (if loaded from file)
	// Environment variables take precedence for Addr/Path, file sets observability
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

	// Merge MCP observability from mosaic config
	mcpCfg.Observability = mosaicLoaded.MCPObservability

	// Load ratchet config if provided
	ratchetConfigPath := config.ResolveRatchetConfigPath(*ratchetConfigFlag)
	ratchetConfig, err := config.LoadRatchetConfig(ratchetConfigPath)
	if err != nil {
		return fmt.Errorf("ratchet config: %w", err)
	}
	mosaicLoaded.RatchetConfig = ratchetConfig

	// Initialize ratchet service if config is provided
	var ratchetSvc ratchetprimary.RatchetService
	var sessionStore ratchetsecondary.SessionStore
	var broadcaster *eventBroadcaster
	if ratchetConfig.Path != "" {
		configLoader := ratchetadapters.NewYAMLConfigLoader()
		tokenStore := ratchetadapters.NewMemoryTokenStore()
		sessionStore = ratchetadapters.NewMemorySessionStore()
		randomGen := ratchetadapters.NewCryptoRandomGenerator()
		clock := ratchetadapters.NewRealClock()

		// Load full ratchet configuration (rules + observability settings)
		configFile, err := os.Open(ratchetConfig.Path) // #nosec G304 - path from flag
		if err != nil {
			return fmt.Errorf("open ratchet config file: %w", err)
		}
		defer func() {
			if cerr := configFile.Close(); cerr != nil {
				log.Warn("close ratchet config file", "err", cerr)
			}
		}()

		ctx := context.Background()
		fullCfg, err := configLoader.LoadConfig(ctx, configFile)
		if err != nil {
			return fmt.Errorf("load ratchet configuration: %w", err)
		}

		// Create base event store
		baseStore := ratchetadapters.NewMemoryEventStore(fullCfg.Observability.RetentionDays)

		// If WebSocket enabled, wrap with broadcaster for real-time streaming
		eventStore := ratchetsecondary.EventStore(baseStore)
		if mcpCfg.Observability.WebSocketEnabled {
			broadcaster = newEventBroadcaster()
			eventStore = ratchetadapters.NewBroadcastingEventStore(baseStore, broadcaster)
			log.Info("WebSocket broadcasting enabled")
		}

		if eventStore != nil {
			log.Info("Ratchet observability enabled", "storage_type", fullCfg.Observability.StorageType)
		}

		ratchetSvc = ratchetservices.NewRatchetServiceWithObservability(
			configLoader,
			tokenStore,
			sessionStore,
			randomGen,
			clock,
			eventStore,
		)

		// Register rules from loaded config
		for _, rule := range fullCfg.Rules {
			if err := ratchetSvc.RegisterRule(ctx, rule); err != nil {
				return fmt.Errorf("register rule for tool %s: %w", rule.Tool, err)
			}
		}

		log.Info("ratchet config loaded",
			"config_file", ratchetConfig.Path,
			"rules_count", len(fullCfg.Rules))
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
	healthSvc := mosaicservices.NewHealthService(engineHealth, version, mosaicLoaded.MVCCRetainCommitsBehindHead)

	cellReader := hexxlastore.NewCellReaderAdapter(live)
	ollamaClient := ollamac.NewClient(ollamaCfg.Base, ollamaCfg.Model)
	cellRetrieval := mosaicservices.NewCellRetrievalService(
		cellReader,
		ollamaembed.NewTextEmbedder(ollamaClient),
		embeddingDimAtBoot,
	)

	contextPackLoader := hexxlastore.NewContextPackAdapter(live)
	contextAssembly := mosaicservices.NewContextAssemblyService(contextPackLoader)

	embeddingANN := hexxlastore.NewEmbeddingANNAdapter(live, ollamaClient)
	embeddingSvc := mosaicservices.NewEmbeddingSearchService(embeddingANN)

	cellWriter := hexxlastore.NewCellWriterAdapter(live, dbCfg.Path, openOpts, mosaicLoaded.DeleteAutoMaintain)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := cellWriter.FlushPostDeleteMaintain(shutdownCtx); err != nil {
			log.Error("flush post-delete maintain", "err", err)
		}
	}()

	runtimeCfg := config.NewMosaicRuntimeConfig(mosaicLoaded.Retention, mosaicLoaded.AllowDeleteCell)
	mutationSvc := mosaicservices.NewCellMutationService(
		cellWriter, ollamaembed.NewTextEmbedder(ollamaClient), embeddingDimAtBoot,
		mosaicservices.WithMosaicRuntime(runtimeCfg),
	)

	seamStore := hexxlastore.NewSeamStoreAdapter(live)
	seamSvc := mosaicservices.NewSeamLifecycleService(seamStore)

	facetEdgeStore := hexxlastore.NewFacetEdgeStoreAdapter(live)
	facetEdgeSvc := mosaicservices.NewFacetEdgeService(facetEdgeStore)
	facetEdgeBrowse := mosaicservices.NewFacetEdgeReadService(facetEdgeStore)

	tagCatalog := hexxlastore.NewTagCatalogAdapter(live)
	tagBrowseSvc := mosaicservices.NewTagCatalogService(tagCatalog)

	// Create ratchet wrapper if ratchet service is available
	var ratchetWrapper *mcpsrv.RatchetWrapper
	if ratchetConfig.Path != "" {
		ratchetWrapper = mcpsrv.NewRatchetWrapper(ratchetSvc, sessionStore, log)
	}

	policyInstructions := config.MCPPolicyInstructions(runtimeCfg, configPathUsed)
	srv := mcpsrv.NewServer("mosaic", version, mcpsrv.ServerInstructions(policyInstructions))
	mcpsrv.RegisterHealthTool(srv, healthSvc, log, retrievalBudget, ratchetWrapper)
	mcpsrv.RegisterCellQueryTool(srv, cellRetrieval, log, retrievalBudget, ratchetWrapper)
	mcpsrv.RegisterCellSearchTool(srv, cellRetrieval, log, retrievalBudget, ratchetWrapper)
	mcpsrv.RegisterEmbeddingSearchTool(srv, embeddingSvc, log, retrievalBudget, ratchetWrapper)
	mcpsrv.RegisterContextPackTool(srv, contextAssembly, log, retrievalBudget, ratchetWrapper)
	mcpsrv.RegisterContextBudgetEstimateTool(srv, log, ratchetWrapper)
	mcpsrv.RegisterCellMutationTools(srv, mutationSvc, runtimeCfg, log, ratchetWrapper)
	mcpsrv.RegisterSeamTools(srv, seamSvc, log, retrievalBudget, ratchetWrapper)
	mcpsrv.RegisterFacetEdgeTools(srv, facetEdgeSvc, log, ratchetWrapper)
	mcpsrv.RegisterFacetEdgeBrowseTools(srv, facetEdgeBrowse, log, retrievalBudget, ratchetWrapper)
	mcpsrv.RegisterTagBrowseTools(srv, tagBrowseSvc, log, retrievalBudget, ratchetWrapper)
	mcpsrv.RegisterPersistencePolicyTool(srv, runtimeCfg, configPathUsed, log, ratchetWrapper)
	mcpsrv.RegisterRetrievalBudgetStatusTool(srv, retrievalBudget, log, ratchetWrapper)
	h := mcpsrv.StreamableHTTPHandler(srv, log)

	mux := http.NewServeMux()
	for _, method := range [...]string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		mux.Handle(method+" "+mcpCfg.Path, h)
	}

	// Observability endpoints (web UI support)
	// GET /observability/stats - aggregate statistics
	// GET /observability/events?session_id=<id> - events for session
	if ratchetSvc != nil && mcpCfg.Observability.HTTPEnabled {
		mux.HandleFunc("GET /observability/stats", func(w http.ResponseWriter, r *http.Request) {
			stats, err := ratchetSvc.GetObservabilityStats(r.Context())
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to get stats: %v", err), http.StatusInternalServerError)
				return
			}
			if stats == nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "observability disabled"})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(stats); err != nil {
				log.Warn("failed to encode stats", "error", err)
			}
		})

		mux.HandleFunc("GET /observability/events", func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			sessionID := ratchetdomain.SessionID(q.Get("session_id"))

			// Build filter from query params
			filter := &ratchetsecondary.EventFilter{}

			// ?event_type=tool_call_failure,token_created (comma-separated)
			if raw := q.Get("event_type"); raw != "" {
				for t := range strings.SplitSeq(raw, ",") {
					filter.EventTypes = append(filter.EventTypes, ratchetdomain.EventType(strings.TrimSpace(t)))
				}
			}

			// ?tool_name=greet,get_user_name (comma-separated)
			if raw := q.Get("tool_name"); raw != "" {
				for t := range strings.SplitSeq(raw, ",") {
					filter.ToolNames = append(filter.ToolNames, ratchetdomain.ToolName(strings.TrimSpace(t)))
				}
			}

			// ?limit=50 (default 100)
			filter.Limit = 100
			if raw := q.Get("limit"); raw != "" {
				if n, err := strconv.Atoi(raw); err == nil && n > 0 {
					filter.Limit = n
				}
			}

			// ?offset=0 (pagination)
			offset := 0
			if raw := q.Get("offset"); raw != "" {
				if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
					offset = n
				}
			}

			// Fetch with limit+offset to allow slicing
			filter.Limit += offset
			events, err := ratchetSvc.GetObservabilityEvents(r.Context(), sessionID, filter)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to get events: %v", err), http.StatusInternalServerError)
				return
			}

			// Apply offset
			if offset > 0 && offset < len(events) {
				events = events[offset:]
			} else if offset >= len(events) {
				events = []*ratchetdomain.Event{}
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(events); err != nil {
				log.Warn("failed to encode events", "error", err)
			}
		})
	}

	// WebSocket streaming endpoint (real-time events)
	// Connect with: wscat -c "ws://localhost:8787/observability/stream?session_id=mosaic-mcp-session"
	if ratchetSvc != nil && mcpCfg.Observability.WebSocketEnabled && broadcaster != nil {
		mux.HandleFunc("GET "+mcpCfg.Observability.WebSocketPath, func(w http.ResponseWriter, r *http.Request) {
			sessionID := ratchetdomain.SessionID(r.URL.Query().Get("session_id"))
			if sessionID == "" {
				http.Error(w, "session_id required", http.StatusBadRequest)
				return
			}

			conn, err := wsUpgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Warn("websocket upgrade failed", "error", err)
				return
			}
			defer func() { _ = conn.Close() }()

			broadcaster.subscribe(sessionID, conn)
			defer broadcaster.unsubscribe(sessionID, conn)

			log.Info("websocket client connected", "session_id", sessionID, "remote_addr", r.RemoteAddr)

			// Send initial confirmation
			if err := conn.WriteJSON(map[string]string{
				"type":       "connected",
				"session_id": string(sessionID),
			}); err != nil {
				log.Warn("failed to send websocket confirmation", "error", err)
				return
			}

			// Keep connection alive and listen for client disconnect
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Warn("websocket error", "error", err)
					}
					break
				}
			}

			log.Info("websocket client disconnected", "session_id", sessionID)
		})

		log.Info("WebSocket streaming enabled", "endpoint", mcpCfg.Observability.WebSocketPath)
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
		if ratchetSvc != nil && mcpCfg.Observability.HTTPEnabled {
			log.Info("Observability endpoints available", "stats", "/observability/stats", "events", "/observability/events")
		}
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
