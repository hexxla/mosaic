package domain

import "time"

// SnapshotSelector selects one retained transaction snapshot; its fields are exclusive.
type SnapshotSelector struct {
	Seq *uint64    `json:"seq,omitempty"`
	Tag string     `json:"tag,omitempty"`
	At  *time.Time `json:"at,omitempty"`
}

// GraphQuery bounds relationship exploration.
type GraphQuery struct {
	Start       AxialCoord  `json:"start"`
	Goal        *AxialCoord `json:"goal,omitempty"`
	Relation    string      `json:"relation,omitempty"`
	MaxHops     int         `json:"max_hops,omitzero"`
	MaxNodes    int         `json:"max_nodes,omitzero"`
	MaxEdgeRows int         `json:"max_edge_rows,omitzero"`
}

// CoordinateResults contains an ordered traversal or path.
type CoordinateResults struct {
	Coords       []AxialCoord `json:"coords"`
	LimitReached bool         `json:"limit_reached,omitzero"`
}

// SpatialQuery selects bounded FOV or Voronoi context without remote callbacks.
type SpatialQuery struct {
	Seeds    []AxialCoord `json:"seeds"`
	Radius   int          `json:"radius,omitzero"`
	MaxCells int          `json:"max_cells,omitzero"`
	Blocked  []AxialCoord `json:"blocked,omitempty"`
}

// SpatialRegion associates cells with their input seed index.
type SpatialRegion struct {
	SeedIndex int       `json:"seed_index"`
	Cells     []CellHit `json:"cells"`
}

// SpatialResults is a bounded set of context regions.
type SpatialResults struct {
	Regions []SpatialRegion `json:"regions"`
}

// RingOccupancy reports geometric capacity and current occupancy.
type RingOccupancy struct {
	Ring     int `json:"ring"`
	Occupied int `json:"occupied"`
	Total    int `json:"total"`
}

// DensityResults describes a bounded neighborhood.
type DensityResults struct {
	Rings []RingOccupancy `json:"rings"`
}

// GridResult contains a small textual lattice diagram.
type GridResult struct {
	Grid string `json:"grid"`
}

// TaxonomyQuery bounds a regional or sampled taxonomy analysis.
type TaxonomyQuery struct {
	Center      AxialCoord `json:"center"`
	Radius      int        `json:"radius,omitzero"`
	MinCount    int        `json:"min_count,omitzero"`
	MaxResults  int        `json:"max_results,omitzero"`
	MaxScanRows int        `json:"max_scan_rows,omitzero"`
}

// TagPairCount counts co-occurrence within the reported cell sample.
type TagPairCount struct {
	A     string `json:"a"`
	B     string `json:"b"`
	Count int    `json:"count"`
}

// TaxonomyResults makes sampling and truncation explicit.
type TaxonomyResults struct {
	Pairs         []TagPairCount `json:"pairs,omitempty"`
	Coords        []AxialCoord   `json:"coords,omitempty"`
	CellsExamined int            `json:"cells_examined"`
	Partial       bool           `json:"partial"`
}

// SnapshotLabel is a named sequence, subject to retention expiry.
type SnapshotLabel struct {
	Label string `json:"label"`
	Seq   uint64 `json:"seq"`
}

// SnapshotLabels lists retained names; tags do not pin history against pruning.
type SnapshotLabels struct {
	Tags      []SnapshotLabel `json:"tags"`
	Truncated bool            `json:"truncated"`
	OldestSeq uint64          `json:"oldest_seq"`
}

// HistoricalCellQuery reads one cell from a selected snapshot.
type HistoricalCellQuery struct {
	Coord    AxialCoord       `json:"coord"`
	Snapshot SnapshotSelector `json:"snapshot"`
}

// HistoryDiffQuery selects a narrow retained-history window.
type HistoryDiffQuery struct {
	FromSeq    uint64 `json:"from_seq"`
	ToSeq      uint64 `json:"to_seq"`
	MaxResults int    `json:"max_results,omitzero"`
}

// HistoryChange contains one retained cell or seam mutation.
type HistoryChange struct {
	Seq       uint64   `json:"seq"`
	Operation string   `json:"operation"`
	Cell      *CellHit `json:"cell,omitempty"`
	Seam      *SeamHit `json:"seam,omitempty"`
}

// HistoryDiffResults is diagnostic retained history, not a complete changefeed.
type HistoryDiffResults struct {
	HistoryPruned bool            `json:"history_pruned"`
	Changes       []HistoryChange `json:"changes"`
	Truncated     bool            `json:"truncated"`
	OldestSeq     uint64          `json:"oldest_seq"`
}

// ChangesQuery bounds physical changefeed records examined, including filtered records.
type ChangesQuery struct {
	AfterSeq       uint64   `json:"after_seq,omitzero"`
	MaxRecords     int      `json:"max_records,omitzero"`
	Operations     []string `json:"operations,omitempty"`
	IncludePayload bool     `json:"include_payload,omitzero"`
}

// ChangeEvent carries opaque encoded identity/payload in base64, never interpreted physical keys.
type ChangeEvent struct {
	Seq           uint64 `json:"seq"`
	Operation     string `json:"operation"`
	WallTime      string `json:"wall_time,omitempty"`
	KeyBase64     string `json:"key_base64"`
	SHA256        string `json:"sha256,omitempty"`
	EncodedBytes  uint32 `json:"encoded_bytes"`
	PayloadBase64 string `json:"payload_base64,omitempty"`
}

// ChangesResults advances across examined records even when filters match none.
type ChangesResults struct {
	Events      []ChangeEvent `json:"events"`
	NextSeq     uint64        `json:"next_seq"`
	MayHaveMore bool          `json:"may_have_more"`
}

// VectorResult exposes an independently stored embedding.
type VectorResult struct {
	Found  bool      `json:"found"`
	Vector []float32 `json:"vector,omitempty"`
	Model  string    `json:"embedding_model,omitempty"`
}

// RebuildIndexQuery places explicit limits on derived index reconstruction.
type RebuildIndexQuery struct {
	MaxVectors     int    `json:"max_vectors,omitzero"`
	MaxMemoryBytes uint64 `json:"max_memory_bytes,omitzero"`
}

// RebuildIndexResult reports completed derived-index work.
type RebuildIndexResult struct {
	Vectors            int    `json:"vectors"`
	Revision           uint64 `json:"revision"`
	EstimatedPeakBytes uint64 `json:"estimated_peak_bytes"`
}

// DiagnosticsResult contains storage and cumulative write counters with explicit units.
type DiagnosticsResult struct {
	Storage   map[string]uint64 `json:"storage"`
	Writes    map[string]uint64 `json:"writes"`
	OldestSeq uint64            `json:"oldest_seq"`
	Model     string            `json:"embedding_model,omitempty"`
}
