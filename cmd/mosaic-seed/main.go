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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/go-llm-project-structure/internal/config"
	ollamac "github.com/sploitzberg/go-llm-project-structure/internal/ollama"
)

// defaultOllamaURL is overridden by MOSAIC_OLLAMA_URL when -ollama is not set (after parsing).
const defaultOllamaURL = "http://127.0.0.1:11434"

const envOllamaURL = "MOSAIC_OLLAMA_URL"
const envEmbedModel = "MOSAIC_EMBED_MODEL"

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbFlag := flag.String("db", "", "path to HexxlaDB file (overrides -name; default: "+config.EnvDBPath+", else "+config.MosaicDefaultRelDBFile+")")
	nameFlag := flag.String("name", "", "base name for the file: <db-dir>/<name>.hexxla (mutually exclusive with -db; directory from -db-dir or "+config.EnvMosaicDBDir+")")
	dbDirFlag := flag.String("db-dir", "", "parent directory when using -name (default: from "+config.EnvMosaicDBDir+" or .tmp)")
	var replaceExisting bool
	flag.BoolVar(&replaceExisting, "force", false, "replace existing file at the chosen path: delete it then seed")
	flag.BoolVar(&replaceExisting, "replace", false, "same as -force")
	ollamaFlag := flag.String("ollama", "", "Ollama base URL (empty: "+envOllamaURL+" or "+defaultOllamaURL+")")
	embedModel := flag.String("embed-model", "", "Ollama embedding model (empty: "+envEmbedModel+" or all-minilm)")
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
	ollamaBase := resolveOllamaURL(*ollamaFlag)
	model := resolveEmbedModel(*embedModel)

	layout, err := config.ParseMosaicDatabaseLayoutFromCLI(*mvcc, *pageSize, *maxVal, *embedDim, *metricStr)
	if err != nil {
		log.Error(err.Error())
		os.Exit(2)
	}

	u, err := url.Parse(ollamaBase)
	if err != nil {
		log.Error("invalid Ollama URL", "err", err, "base", ollamaBase)
		os.Exit(2)
	}
	oc := ollamac.NewClient(u, model)
	oc.HTTP = &http.Client{Timeout: ollamac.DefaultEmbedTimeout}

	if err := run(log, dbPath, replaceExisting, oc, *dbPassphrase, layout); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func resolveOllamaURL(flagValue string) string {
	p := strings.TrimSpace(flagValue)
	if p != "" {
		return p
	}
	p = strings.TrimSpace(os.Getenv(envOllamaURL))
	if p != "" {
		return p
	}
	return defaultOllamaURL
}

func resolveEmbedModel(flagValue string) string {
	p := strings.TrimSpace(flagValue)
	if p != "" {
		return p
	}
	p = strings.TrimSpace(os.Getenv(envEmbedModel))
	if p != "" {
		return p
	}
	return "all-minilm"
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
