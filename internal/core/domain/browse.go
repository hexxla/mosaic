package domain

// BrowseQuery resumes deterministic nearest-ring enumeration at one snapshot.
// Ring and Offset are application continuation state, never physical byte keys.
type BrowseQuery struct {
	Center         AxialCoord `json:"center"`
	Radius         int        `json:"radius,omitzero"`
	MaxCells       int        `json:"max_cells,omitzero"`
	MaxProbes      int        `json:"max_probes,omitzero"`
	MaxBytes       int        `json:"max_bytes,omitzero"`
	IncludeContent bool       `json:"include_content,omitzero"`
	SnapshotSeq    *uint64    `json:"snapshot_seq,omitempty"`
	Ring           int        `json:"-"`
	Offset         int        `json:"-"`
}

// BrowsePage reports progress across empty coordinates as well as live cells.
type BrowsePage struct {
	Cells         []CellHit    `json:"cells"`
	DeferredCells []AxialCoord `json:"deferred_cells,omitempty"`
	SnapshotSeq   uint64       `json:"snapshot_seq"`
	Probes        int          `json:"probes"`
	Done          bool         `json:"done"`
	NextCursor    string       `json:"next_cursor,omitempty"`
	NextRing      int          `json:"-"`
	NextOffset    int          `json:"-"`
}

// CellChunkQuery reads raw content or a complete native CellRecord JSON document.
// Base64 preserves arbitrary byte boundaries and avoids Unicode replacement.
type CellChunkQuery struct {
	Coord    AxialCoord       `json:"coord"`
	Snapshot SnapshotSelector `json:"snapshot"`
	Format   string           `json:"format,omitempty"`
	Offset   int              `json:"offset,omitzero"`
	MaxBytes int              `json:"max_bytes,omitzero"`
}

// CellChunkResult is a resumable byte slice of a snapshot-pinned record.
type CellChunkResult struct {
	Found      bool   `json:"found"`
	DataBase64 string `json:"data_base64,omitempty"`
	TotalBytes int    `json:"total_bytes"`
	NextOffset int    `json:"next_offset"`
	SHA256     string `json:"sha256,omitempty"`
	Done       bool   `json:"done"`
}
