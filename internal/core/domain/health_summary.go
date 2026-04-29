package domain

// HealthSummary is an engine-agnostic snapshot from a database health scan.
type HealthSummary struct {
	// MosaicVersion is the mosaic-mcp binary version (from the application layer).
	MosaicVersion string `json:"mosaic_version" jsonschema:"mosaic MCP server version string"`

	// DatabaseLayout is read from the open file header (not from the HealthCheck scan).
	DatabaseLayout DatabaseLayout `json:"database_layout" jsonschema:"page size, value cap, embedding config"`

	// IntegrityOK is true when the health scan reported no tag/source index errors and no orphan seams.
	IntegrityOK bool `json:"integrity_ok" jsonschema:"true if index and seam reference checks are clean"`

	CellCount         int      `json:"cell_count" jsonschema:"number of visible cells"`
	SeamCount         int      `json:"seam_count" jsonschema:"number of visible seams"`
	SeamsResolved     int      `json:"seams_resolved" jsonschema:"seams with resolution recorded"`
	SeamsUnresolved   int      `json:"seams_unresolved" jsonschema:"seams still unresolved"`
	OrphanSeamIDs     []string `json:"orphan_seam_ids,omitempty" jsonschema:"seam IDs whose endpoint cells are missing"`
	TagIndexErrors    int      `json:"tag_index_errors" jsonschema:"tag secondary index inconsistencies"`
	SourceIndexErrors int      `json:"source_index_errors" jsonschema:"source secondary index inconsistencies"`

	MVCC MVCCSnapshot `json:"mvcc" jsonschema:"MVCC storage counters when enabled"`

	// Warnings lists non-fatal diagnostic messages from the engine.
	Warnings []string `json:"warnings,omitempty" jsonschema:"human-readable warnings"`
}

// MVCCSnapshot mirrors engine MVCC counters relevant to operators.
type MVCCSnapshot struct {
	CommitSeq     uint64 `json:"commit_seq" jsonschema:"current commit sequence"`
	VersionedRows int64  `json:"versioned_rows" jsonschema:"versioned cell rows"`
	LogicalCells  int64  `json:"logical_cells" jsonschema:"logical cell count in MVCC model"`
}

// DatabaseLayout summarizes persisted parameters from the HexxlaDB file header (operator-facing).
type DatabaseLayout struct {
	PageSize           uint32 `json:"page_size_bytes" jsonschema:"B-tree page size"`
	MaxValueBytes      uint32 `json:"max_value_bytes" jsonschema:"max encoded value size"`
	EmbeddingDimension uint16 `json:"embedding_dimension" jsonschema:"fixed embedding vector dimension (0 if disabled)"`
	EmbeddingMetric    string `json:"embedding_metric,omitempty" jsonschema:"distance metric name when embeddings enabled"`
}
