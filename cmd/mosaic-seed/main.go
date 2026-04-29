// Command mosaic-seed creates or refreshes a HexxlaDB file with conversational cells and Ollama
// embeddings — the same ingestion pattern as github.com/hexxla/hexxladb/examples/llm_context_engine
// (PutCell + PutEmbedding per turn) overlaid on coordinates from conversational_memory's spiral grid.
//
// Requires a running Ollama with the embedding model (default: all-minilm). Use the resulting file
// with MOSAIC_DB_PATH when running cmd/mosaic-mcp.
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

	ollamac "github.com/sploitzberg/go-llm-project-structure/internal/ollama"
)

// defaultDBPath mirrors hexxladb demos (.tmp under the project root, gitignored — not system /tmp).
const defaultDBPath = ".tmp/mosaic-seed.hexxla"

const envDBPath = "MOSAIC_DB_PATH"

// defaultOllamaURL is overridden by MOSAIC_OLLAMA_URL when -ollama is not set (after parsing).
const defaultOllamaURL = "http://127.0.0.1:11434"

const envOllamaURL = "MOSAIC_OLLAMA_URL"
const envEmbedModel = "MOSAIC_EMBED_MODEL"

// Must match [hexxladb.Options.EmbeddingDimension] for this seed Open options.
const embeddingDimAllMiniLM = 384

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbFlag := flag.String("db", "", "path to HexxlaDB file (default: MOSAIC_DB_PATH, else "+defaultDBPath+")")
	force := flag.Bool("force", false, "remove existing database file(s) and re-seed")
	ollamaFlag := flag.String("ollama", "", "Ollama base URL (empty: "+envOllamaURL+" or "+defaultOllamaURL+")")
	embedModel := flag.String("embed-model", "", "Ollama embedding model (empty: "+envEmbedModel+" or all-minilm)")

	flag.Parse()

	dbPath := resolveDBPath(*dbFlag)
	ollamaBase := resolveOllamaURL(*ollamaFlag)
	model := resolveEmbedModel(*embedModel)

	u, err := url.Parse(ollamaBase)
	if err != nil {
		log.Error("invalid Ollama URL", "err", err, "base", ollamaBase)
		os.Exit(2)
	}
	oc := ollamac.NewClient(u, model)
	oc.HTTP = &http.Client{Timeout: ollamac.DefaultEmbedTimeout}

	if err := run(log, dbPath, *force, oc); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func resolveDBPath(flagValue string) string {
	p := strings.TrimSpace(flagValue)
	if p != "" {
		return filepath.Clean(p)
	}
	p = strings.TrimSpace(os.Getenv(envDBPath))
	if p != "" {
		return filepath.Clean(p)
	}
	return filepath.Clean(defaultDBPath)
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
		log.Info("database already exists; skipping seed (use -force to replace)", "path", dbPath)
		return nil
	}

	if force {
		_ = os.Remove(dbPath)
		_ = os.Remove(dbPath + "-wal")
	}

	if !exists {
		_ = os.Remove(dbPath + "-wal")
	}

	// Matches examples/llm_context_engine Open (384-d cosine; MVCC enabled for new DBs).
	opts := &hexxladb.Options{
		EnableMVCC:    true,
		PageSize:      65536,
		MaxValueBytes: 16384,

		EmbeddingDimension: embeddingDimAllMiniLM,
		DistanceMetric:     hexxladb.DistanceCosine,
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

	if len(seedConversation) == 0 {
		log.Info("no seed corpus; database opened empty", "path", dbPath)
		return nil
	}

	log.Info("seeding corpus with embeddings", "turns", len(seedConversation))

	for i := range seedConversation {
		msg := seedConversation[i]

		vec, err := oc.Embed(ctx, msg.content, embeddingDimAllMiniLM)
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
		"embedding_dim", embeddingDimAllMiniLM,
		"sessionID", sessionID,
	)
	return nil
}
