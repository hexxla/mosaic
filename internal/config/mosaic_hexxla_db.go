package config

import (
	"fmt"
	"math"
	"strings"

	"github.com/hexxla/hexxladb"
)

// MosaicEmbeddingDimensionAllMiniLM is the default embedding width for Mosaic tooling when using
// Ollama all-MiniLM-L6-v2 family models (matches historical mosaic-seed defaults).
const MosaicEmbeddingDimensionAllMiniLM = 384

// Defaults for new Mosaic-created HexxlaDB files (MVCC, 4 KiB pages, 384-d cosine embeddings).
// Plaintext creation uses format v2; official encryption creates authenticated format v3.
const (
	MosaicDefaultPageSize      uint32 = 4096
	MosaicDefaultMaxValueBytes uint32 = 16384
)

// MosaicDatabaseLayout describes persisted-on-create HexxlaDB options shared by mosaic-create-db and mosaic-seed.
// Encryption is applied separately via [ApplyHexxlaEncryption].
type MosaicDatabaseLayout struct {
	PageSize           uint32
	MaxValueBytes      uint32
	EmbeddingDimension uint16
	DistanceMetric     hexxladb.DistanceMetric
}

// DefaultMosaicDatabaseLayout returns the shared database creation defaults.
func DefaultMosaicDatabaseLayout() MosaicDatabaseLayout {
	return MosaicDatabaseLayout{
		PageSize:           MosaicDefaultPageSize,
		MaxValueBytes:      MosaicDefaultMaxValueBytes,
		EmbeddingDimension: MosaicEmbeddingDimensionAllMiniLM,
		DistanceMetric:     hexxladb.DistanceCosine,
	}
}

// BuildMosaicCreateOptions applies the same credential rules as database open.
func BuildMosaicCreateOptions(layout MosaicDatabaseLayout, credentials HexxlaOpenParams) (*hexxladb.Options, error) {
	opts := &hexxladb.Options{
		PageSize: layout.PageSize, MaxValueBytes: layout.MaxValueBytes,
		EmbeddingDimension: layout.EmbeddingDimension, DistanceMetric: layout.DistanceMetric,
	}
	if err := ApplyHexxlaEncryption(opts, credentials); err != nil {
		return nil, err
	}
	return opts, nil
}

// ParseMosaicDatabaseLayoutFromCLI builds [MosaicDatabaseLayout] from mosaic-create-db / mosaic-seed flags.
// page-size, max-value-bytes, embedding-dim use flag defaults matching [DefaultMosaicDatabaseLayout].
func ParseMosaicDatabaseLayoutFromCLI(pageSize, maxVal, embedDim uint, metricStr string) (MosaicDatabaseLayout, error) {
	metric, err := ParseDistanceMetricName(metricStr)
	if err != nil {
		return MosaicDatabaseLayout{}, err
	}
	if embedDim > 0xffff {
		return MosaicDatabaseLayout{}, fmt.Errorf("config: embedding-dim too large: %d", embedDim)
	}
	if pageSize > math.MaxUint32 || maxVal > math.MaxUint32 {
		return MosaicDatabaseLayout{}, fmt.Errorf("config: page-size or max-value-bytes too large")
	}
	return MosaicDatabaseLayout{
		PageSize:           uint32(pageSize), //nolint:gosec // G115: bounded by check above
		MaxValueBytes:      uint32(maxVal),   //nolint:gosec // G115: bounded by check above
		EmbeddingDimension: uint16(embedDim),
		DistanceMetric:     metric,
	}, nil
}

// ParseDistanceMetricName maps CLI strings (cosine, l2, dot) to HexxlaDB embedding metrics.
func ParseDistanceMetricName(s string) (hexxladb.DistanceMetric, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "cosine":
		return hexxladb.DistanceCosine, nil
	case "l2", "euclidean":
		return hexxladb.DistanceL2, nil
	case "dot", "dotproduct":
		return hexxladb.DistanceDotProduct, nil
	default:
		return 0, fmt.Errorf("config: distance metric must be cosine, l2, or dot, got %q", s)
	}
}
