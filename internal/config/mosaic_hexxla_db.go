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

// Defaults for new Mosaic-created HexxlaDB files (MVCC v2, 64 KiB pages, 384-d cosine embeddings).
const (
	MosaicDefaultPageSize      uint32 = 65536
	MosaicDefaultMaxValueBytes uint32 = 16384
)

// MosaicDatabaseLayout describes persisted-on-create HexxlaDB options shared by mosaic-create-db and mosaic-seed.
// Encryption is applied separately via [ApplyHexxlaEncryption].
type MosaicDatabaseLayout struct {
	EnableMVCC         bool
	PageSize           uint32
	MaxValueBytes      uint32
	EmbeddingDimension uint16
	DistanceMetric     hexxladb.DistanceMetric
}

// DefaultMosaicDatabaseLayout matches historical Mosaic defaults (same as [MosaicNewDatabaseOptions]).
func DefaultMosaicDatabaseLayout() MosaicDatabaseLayout {
	return MosaicDatabaseLayout{
		EnableMVCC:         true,
		PageSize:           MosaicDefaultPageSize,
		MaxValueBytes:      MosaicDefaultMaxValueBytes,
		EmbeddingDimension: MosaicEmbeddingDimensionAllMiniLM,
		DistanceMetric:     hexxladb.DistanceCosine,
	}
}

// NewMosaicDatabaseOptions builds [hexxladb.Options] for creating a new database file.
// Invalid combinations are rejected by [hexxladb.Open], not here.
func NewMosaicDatabaseOptions(layout MosaicDatabaseLayout) *hexxladb.Options {
	return &hexxladb.Options{
		EnableMVCC:         layout.EnableMVCC,
		PageSize:           layout.PageSize,
		MaxValueBytes:      layout.MaxValueBytes,
		EmbeddingDimension: layout.EmbeddingDimension,
		DistanceMetric:     layout.DistanceMetric,
	}
}

// MosaicNewDatabaseOptions returns [hexxladb.Options] for creating a new Mosaic-compatible HexxlaDB file
// using [DefaultMosaicDatabaseLayout]. Use [NewMosaicDatabaseOptions] when overriding layout via CLI flags.
func MosaicNewDatabaseOptions() *hexxladb.Options {
	return NewMosaicDatabaseOptions(DefaultMosaicDatabaseLayout())
}

// ParseMosaicDatabaseLayoutFromCLI builds [MosaicDatabaseLayout] from mosaic-create-db / mosaic-seed flags.
// page-size, max-value-bytes, embedding-dim use flag defaults matching [DefaultMosaicDatabaseLayout].
func ParseMosaicDatabaseLayoutFromCLI(mvcc bool, pageSize, maxVal, embedDim uint, metricStr string) (MosaicDatabaseLayout, error) {
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
		EnableMVCC:         mvcc,
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
