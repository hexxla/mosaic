// Command mosaic-seed builds a HexxlaDB candidate with conversational cells and
// Ollama embeddings. Each turn and its embedding share one transaction.
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
	"os"
	"strings"
	"time"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/adapter/secondary/hexxlastore"
	"github.com/sploitzberg/mosaic/internal/config"
	ollamac "github.com/sploitzberg/mosaic/internal/ollama"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	policyFlag := flag.String("policy", "", "path to Mosaic config YAML for Ollama and database credentials")
	dbFlag := flag.String("db", "", "path to HexxlaDB file (overrides -name; default: "+config.EnvDBPath+", else "+config.MosaicDefaultRelDBFile+")")
	nameFlag := flag.String("name", "", "base name for the file: <db-dir>/<name>.hexxla (mutually exclusive with -db; directory from -db-dir or "+config.EnvMosaicDBDir+")")
	dbDirFlag := flag.String("db-dir", "", "parent directory when using -name (default: from "+config.EnvMosaicDBDir+" or .tmp)")
	var replaceExisting bool
	flag.BoolVar(&replaceExisting, "force", false, "retired; use a new candidate path")
	flag.BoolVar(&replaceExisting, "replace", false, "retired; use a new candidate path")
	ollamaFlag := flag.String("ollama", "", "Ollama base URL (overrides policy YAML then "+config.EnvOllamaURL+")")
	embedModel := flag.String("embed-model", "", "embeddings model (overrides policy YAML then "+config.EnvEmbedModel+")")
	dbPassphrase := flag.String("db-passphrase", "", "optional HexxlaDB encryption passphrase (overrides "+config.EnvDBPassphrase+")")

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
	credentials := config.HexxlaOpenParams{FlagPassphrase: *dbPassphrase}
	configPath := config.ResolveMosaicConfigPath(*policyFlag)
	if strings.TrimSpace(configPath) != "" {
		loaded, err := config.LoadMosaicConfigFromFile(configPath)
		if err != nil {
			log.Error(err.Error())
			os.Exit(2)
		}
		credentials.YAMLPassphrase = loaded.DatabasePassphrase
		ollIn.YAMLBaseURL = loaded.OllamaBaseURL
		ollIn.YAMLEmbedModel = loaded.OllamaEmbedModel
	}
	ollamaCfg, err := config.ResolveOllama(ollIn)
	if err != nil {
		log.Error(err.Error())
		os.Exit(2)
	}

	layout, err := config.ParseMosaicDatabaseLayoutFromCLI(*pageSize, *maxVal, *embedDim, *metricStr)
	if err != nil {
		log.Error(err.Error())
		os.Exit(2)
	}

	oc := ollamac.NewClient(ollamaCfg.Base, ollamaCfg.Model)

	if err := run(log, dbPath, replaceExisting, oc, credentials, layout); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func run(
	log *slog.Logger,
	dbPath string,
	force bool,
	oc *ollamac.Client,
	credentials config.HexxlaOpenParams,
	layout config.MosaicDatabaseLayout,
) error {
	ctx := context.Background()

	if force {
		return fmt.Errorf("in-place replacement is retired; seed a candidate at a new path and switch the database path offline")
	}
	if _, err := os.Lstat(dbPath); err == nil {
		log.Info("database file already exists; skipping seed", "path", dbPath)
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat database: %w", err)
	}

	opts, err := config.BuildMosaicCreateOptions(layout, credentials)
	if err != nil {
		return fmt.Errorf("database encryption options: %w", err)
	}

	if err := oc.Ping(ctx); err != nil {
		return fmt.Errorf("ollama: %w", err)
	}
	log.Info("ollama reachable", "url", oc.Base.String(), "embed_model", oc.Model)
	sessionID := fmt.Sprintf("mosaic-seed-%d", time.Now().UnixNano())
	embedDim := int(layout.EmbeddingDimension)
	if err := hexxlastore.CreateCandidate(ctx, dbPath, opts, func(db *hexxladb.DB) error {
		return seedDatabase(ctx, log, db, oc, sessionID, embedDim)
	}); err != nil {
		return err
	}
	log.Info("seeded HexxlaDB", "path", dbPath, "turns", len(seedConversation), "embedding_dim", embedDim, "sessionID", sessionID)
	return nil
}

func seedDatabase(ctx context.Context, log *slog.Logger, db *hexxladb.DB, oc *ollamac.Client, sessionID string, embedDim int) error {
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

	return nil
}
