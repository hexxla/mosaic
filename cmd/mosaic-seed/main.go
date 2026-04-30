// Command mosaic-seed creates or refreshes a HexxlaDB file with conversational cells and Ollama
// embeddings — the same ingestion pattern as github.com/hexxla/hexxladb/examples/llm_context_engine
// (PutCell + PutEmbedding per turn) overlaid on coordinates from conversational_memory's spiral grid.
//
// Requires a running Ollama with the embedding model (default: all-minilm).
// Path resolution matches cmd/mosaic-create-db — see [config.ResolveMosaicDBPath].
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/config"
	ollamac "github.com/sploitzberg/mosaic/internal/ollama"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	policyFlag := flag.String("policy", "", "path to Mosaic config YAML (optional ollama: base_url, embed_model — override "+config.EnvOllamaURL+" / "+config.EnvEmbedModel+" when CLI -ollama / -embed-model are empty)")
	dbFlag := flag.String("db", "", "path to HexxlaDB file (overrides -name; default: "+config.EnvDBPath+", else "+config.MosaicDefaultRelDBFile+")")
	nameFlag := flag.String("name", "", "base name for the file: <db-dir>/<name>.hexxla (mutually exclusive with -db; directory from -db-dir or "+config.EnvMosaicDBDir+")")
	dbDirFlag := flag.String("db-dir", "", "parent directory when using -name (default: from "+config.EnvMosaicDBDir+" or .tmp)")
	var replaceExisting bool
	flag.BoolVar(&replaceExisting, "force", false, "replace existing file at the chosen path: delete it then seed")
	flag.BoolVar(&replaceExisting, "replace", false, "same as -force")
	ollamaFlag := flag.String("ollama", "", "Ollama base URL (overrides policy YAML then "+config.EnvOllamaURL+")")
	embedModel := flag.String("embed-model", "", "embeddings model (overrides policy YAML then "+config.EnvEmbedModel+")")
	dbPassphrase := flag.String("db-passphrase", "", "optional HexxlaDB encryption passphrase (overrides "+config.EnvDBPassphrase+")")

	mvcc := flag.Bool("mvcc", true, "enable MVCC (format v2) for a new database")
	pageSize := flag.Uint("page-size", uint(config.MosaicDefaultPageSize), "page size for new file (4096, 8192, 16384, or 65536)")
	maxVal := flag.Uint("max-value-bytes", uint(config.MosaicDefaultMaxValueBytes), "max encoded value size per cell")
	embedDim := flag.Uint("embedding-dim", uint(config.MosaicEmbeddingDimensionAllMiniLM), "embedding vector width (must match the embed model output)")
	metricStr := flag.String("distance-metric", "cosine", "embedding distance: cosine, l2, or dot")

	flag.Parse()

	dbPath, err := config.ResolveMosaicDBPath(config.MosaicDBPathInput{
		DBFlag:    *dbFlag,
		NameFlag:  *nameFlag,
		DBDirFlag: *dbDirFlag,
	}, true)
	if err != nil {
		log.Error(err.Error())
		os.Exit(2)
	}
	ollIn := config.OllamaResolveInput{
		FlagBaseURL:    *ollamaFlag,
		FlagEmbedModel: *embedModel,
	}
	configPath := config.ResolveMosaicConfigPath(*policyFlag)
	if strings.TrimSpace(configPath) != "" {
		loaded, err := config.LoadMosaicConfigFromFile(configPath)
		if err != nil {
			log.Error(err.Error())
			os.Exit(2)
		}
		ollIn.YAMLBaseURL = loaded.OllamaBaseURL
		ollIn.YAMLEmbedModel = loaded.OllamaEmbedModel
	}
	ollamaCfg, err := config.ResolveOllama(ollIn)
	if err != nil {
		log.Error(err.Error())
		os.Exit(2)
	}

	layout, err := config.ParseMosaicDatabaseLayoutFromCLI(*mvcc, *pageSize, *maxVal, *embedDim, *metricStr)
	if err != nil {
		log.Error(err.Error())
		os.Exit(2)
	}

	oc := ollamac.NewClient(ollamaCfg.Base, ollamaCfg.Model)
	oc.HTTP = &http.Client{Timeout: ollamac.DefaultEmbedTimeout}

	if err := run(log, dbPath, replaceExisting, oc, *dbPassphrase, layout); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func run(
	log *slog.Logger,
	dbPath string,
	force bool,
	oc *ollamac.Client,
	dbPassphraseFlag string,
	layout config.MosaicDatabaseLayout,
) error {
	ctx := context.Background()

	if err := oc.Ping(ctx); err != nil {
		return fmt.Errorf("ollama: %w", err)
	}
	log.Info("ollama reachable", "url", oc.Base.String(), "embed_model", oc.Model)

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o750); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	exists := false
	if _, err := os.Stat(dbPath); err == nil {
		exists = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat database: %w", err)
	}

	if exists && !force {
		log.Info("database file already exists; skipping seed — use a different -db or -name, or pass -replace / -force to overwrite",
			"path", dbPath)
		return nil
	}

	if force && exists {
		log.Info("replacing existing database file", "path", dbPath)
	}

	if force {
		_ = os.Remove(dbPath)
		_ = os.Remove(dbPath + "-wal")
	}

	if !exists {
		_ = os.Remove(dbPath + "-wal")
	}

	opts := config.NewMosaicDatabaseOptions(layout)
	if err := config.ApplyHexxlaEncryption(opts, config.HexxlaOpenParams{
		FlagPassphrase: dbPassphraseFlag,
	}); err != nil {
		return fmt.Errorf("database encryption options: %w", err)
	}

	db, err := hexxladb.Open(dbPath, opts)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			log.Error("close database", "err", cerr)
		}
	}()

	sessionID := fmt.Sprintf("mosaic-seed-%d", time.Now().UnixNano())
	embedDim := int(layout.EmbeddingDimension)

	if len(seedConversation) == 0 {
		log.Info("no seed corpus; database opened empty", "path", dbPath)
		return nil
	}

	log.Info("seeding corpus with embeddings", "turns", len(seedConversation), "embedding_dim", embedDim)

	for i := range seedConversation {
		msg := seedConversation[i]

		vec, err := oc.Embed(ctx, msg.content, embedDim)
		if err != nil {
			return fmt.Errorf("embed turn %d: %w", i, err)
		}

		pk, err := hexxladb.Pack(spiralCoord(i))
		if err != nil {
			return fmt.Errorf("pack coord index %d: %w", i, err)
		}

		err = db.Update(func(tx *hexxladb.Tx) error {
			switch msg.role {
			case "user":
				rec := hexxladb.NewUserMessageCell(pk, msg.content, sessionID, 1.0)
				rec.Tags = append(rec.Tags, msg.tags...)
				if err := tx.PutCell(ctx, rec); err != nil {
					return fmt.Errorf("put cell index %d: %w", i, err)
				}
			case "assistant":
				rec := hexxladb.NewAssistantResponseCell(pk, msg.content, sessionID, 1.0)
				rec.Tags = append(rec.Tags, msg.tags...)
				if err := tx.PutCell(ctx, rec); err != nil {
					return fmt.Errorf("put cell index %d: %w", i, err)
				}
			default:
				return fmt.Errorf("unknown role %q at index %d", msg.role, i)
			}
			if err := tx.PutEmbedding(pk, vec); err != nil {
				return fmt.Errorf("put embedding index %d: %w", i, err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("seed turn %d: %w", i, err)
		}

		if (i+1)%4 == 0 || i+1 == len(seedConversation) {
			log.Info("embedding progress", "done", i+1, "total", len(seedConversation))
		}
	}

	log.Info("seeded HexxlaDB",
		"path", dbPath,
		"turns", len(seedConversation),
		"embedding_dim", embedDim,
		"sessionID", sessionID,
	)
	return nil
}
