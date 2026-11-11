// Command mosaic-create-db creates an empty Mosaic-compatible HexxlaDB file without Ollama.
//
// At-rest encryption (optional) is applied before open via [config.ApplyHexxlaEncryption]: passphrase
// from -db-passphrase, MOSAIC_DB_PASSPHRASE, YAML database.passphrase (-policy / MOSAIC_POLICY_FILE),
// or MOSAIC_DB_ENCRYPTION_KEY_HEX (raw key; mutually exclusive with passphrase sources).
//
// Database path: -db (full path) or -name (file <db-dir>/<name>.hexxla) — see [config.ResolveMosaicDBPath].
// Layout flags default to the same values as historical mosaic-seed; omit a flag to keep that default.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/hexxla/mosaic/internal/adapter/secondary/hexxlastore"
	"github.com/hexxla/mosaic/internal/config"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	policyFlag := flag.String("policy", "", "path to Mosaic config YAML (optional; for database.passphrase — overrides "+config.EnvPolicyFile+")")
	dbFlag := flag.String("db", "", "path to HexxlaDB file (overrides -name; default: "+config.EnvDBPath+", else "+config.MosaicDefaultRelDBFile+")")
	nameFlag := flag.String("name", "", "base name for the file: <db-dir>/<name>.hexxla (mutually exclusive with -db; directory from -db-dir or "+config.EnvMosaicDBDir+")")
	dbDirFlag := flag.String("db-dir", "", "parent directory when using -name (default: from "+config.EnvMosaicDBDir+" or .tmp)")
	var replaceExisting bool
	flag.BoolVar(&replaceExisting, "force", false, "retired; use a new candidate path")
	flag.BoolVar(&replaceExisting, "replace", false, "retired; use a new candidate path")
	dbPassphrase := flag.String("db-passphrase", "", "optional HexxlaDB encryption passphrase (overrides "+config.EnvDBPassphrase+")")

	pageSize := flag.Uint("page-size", uint(config.MosaicDefaultPageSize), "page size for new file (4096, 8192, 16384, or 65536)")
	maxVal := flag.Uint("max-value-bytes", uint(config.MosaicDefaultMaxValueBytes), "max encoded value size per cell")
	embedDim := flag.Uint("embedding-dim", uint(config.MosaicEmbeddingDimensionAllMiniLM), "embedding vector width (must match your embedding model when using MCP)")
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

	layout, err := config.ParseMosaicDatabaseLayoutFromCLI(*pageSize, *maxVal, *embedDim, *metricStr)
	if err != nil {
		log.Error(err.Error())
		os.Exit(2)
	}

	if err := run(log, dbPath, replaceExisting, *policyFlag, *dbPassphrase, layout); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func run(log *slog.Logger, dbPath string, force bool, policyFlag, dbPassphraseFlag string, layout config.MosaicDatabaseLayout) error {
	if force {
		return fmt.Errorf("in-place replacement is retired; create a candidate at a new path and switch the database path offline")
	}

	configPath := config.ResolveMosaicConfigPath(policyFlag)
	var yamlPass string
	if strings.TrimSpace(configPath) != "" {
		loaded, err := config.LoadMosaicConfigFromFile(configPath)
		if err != nil {
			return fmt.Errorf("mosaic config: %w", err)
		}
		yamlPass = loaded.DatabasePassphrase
	}

	opts, err := config.BuildMosaicCreateOptions(layout, config.HexxlaOpenParams{
		FlagPassphrase: dbPassphraseFlag,
		YAMLPassphrase: yamlPass,
	})
	if err != nil {
		return fmt.Errorf("database encryption options: %w", err)
	}

	if err := hexxlastore.CreateCandidate(context.Background(), dbPath, opts, nil); err != nil {
		return err
	}

	log.Info("created empty Mosaic HexxlaDB",
		"path", dbPath,
		"embedding_dim", layout.EmbeddingDimension,
		"page_size", layout.PageSize,
	)
	return nil
}
