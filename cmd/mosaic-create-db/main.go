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
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/config"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	policyFlag := flag.String("policy", "", "path to Mosaic config YAML (optional; for database.passphrase — overrides "+config.EnvPolicyFile+")")
	dbFlag := flag.String("db", "", "path to HexxlaDB file (overrides -name; default: "+config.EnvDBPath+", else "+config.MosaicDefaultRelDBFile+")")
	nameFlag := flag.String("name", "", "base name for the file: <db-dir>/<name>.hexxla (mutually exclusive with -db; directory from -db-dir or "+config.EnvMosaicDBDir+")")
	dbDirFlag := flag.String("db-dir", "", "parent directory when using -name (default: from "+config.EnvMosaicDBDir+" or .tmp)")
	var replaceExisting bool
	flag.BoolVar(&replaceExisting, "force", false, "replace existing file at the chosen path: delete it then create")
	flag.BoolVar(&replaceExisting, "replace", false, "same as -force")
	dbPassphrase := flag.String("db-passphrase", "", "optional HexxlaDB encryption passphrase (overrides "+config.EnvDBPassphrase+")")

	mvcc := flag.Bool("mvcc", true, "enable MVCC (format v2) for a new database")
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

	layout, err := config.ParseMosaicDatabaseLayoutFromCLI(*mvcc, *pageSize, *maxVal, *embedDim, *metricStr)
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
	configPath := config.ResolveMosaicConfigPath(policyFlag)
	var yamlPass string
	if strings.TrimSpace(configPath) != "" {
		loaded, err := config.LoadMosaicConfigFromFile(configPath)
		if err != nil {
			return fmt.Errorf("mosaic config: %w", err)
		}
		yamlPass = loaded.DatabasePassphrase
	}

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
		return fmt.Errorf("database file already exists at %q — use a different -db or -name, or pass -replace / -force to overwrite", dbPath)
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
		YAMLPassphrase: yamlPass,
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

	log.Info("created empty Mosaic HexxlaDB",
		"path", dbPath,
		"embedding_dim", db.EmbeddingDimension(),
		"mvcc", layout.EnableMVCC,
		"page_size", layout.PageSize,
	)
	return nil
}
