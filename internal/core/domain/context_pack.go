package domain

// LoadContextPackCommand selects HexxlaDB context candidates and Mosaic budgeting.
type LoadContextPackCommand struct {
	Seeds []AxialCoord

	MaxRing  int
	MaxCells int // caps candidate pool; 0 uses the engine default (256)
	// MaxBudgetBytes is the UTF-8 byte ceiling; zero selects the 4096-byte default.
	MaxBudgetBytes int

	// SessionByteLimit further bounds the pack when MCP session budgeting is enabled.
	SessionByteLimit int

	FilterSuperseded bool
	IncludeSeams     bool
	IncludeFacetText bool
	Explain          bool
}

// ContextPackStatsDTO mirrors assembly stats for MCP JSON.
type ContextPackStatsDTO struct {
	CandidatesScanned int `json:"candidates_scanned"`
	CellsEvicted      int `json:"cells_evicted"`
	MaxRingUsed       int `json:"max_ring_used"`
}

// ContextPackCell is a slim projection of a cell in an assembled pack.
type ContextPackCell struct {
	Coord      AxialCoord `json:"coord"`
	RawContent string     `json:"raw_content,omitempty"`
	Tags       []string   `json:"tags,omitempty"`
	SourceID   string     `json:"source_id,omitempty"`
	Confidence float64    `json:"confidence"`
	FacetText  []string   `json:"facet_text,omitempty"`

	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	ValidFrom string `json:"valid_from,omitempty"`
	ValidTo   string `json:"valid_to,omitempty"`

	// BudgetBytes and BudgetRing carry adapter metadata for Mosaic's assembly
	// policy. They are intentionally absent from MCP responses.
	BudgetBytes int `json:"-"`
	BudgetRing  int `json:"-"`
}

// ContextSeamSummary is a minimal seam projection for tooling.
type ContextSeamSummary struct {
	ID               string `json:"id"`
	SeamType         string `json:"seam_type"`
	Reason           string `json:"reason,omitempty"`
	ResolutionStatus string `json:"resolution_status,omitempty"`
}

// ContextPackResponse is the MCP result for Mosaic context assembly.
type ContextPackResponse struct {
	Cells          []ContextPackCell    `json:"cells"`
	TotalBytes     int                  `json:"total_bytes"`
	Stats          ContextPackStatsDTO  `json:"stats"`
	Seams          []ContextSeamSummary `json:"seams,omitempty"`
	Explanations   []string             `json:"explanations,omitempty"`
	RetrievalHint  string               `json:"retrieval_hint,omitempty"`
	SeedCount      int                  `json:"seed_count"`
	MaxRingApplied int                  `json:"max_ring_applied"`
	MaxBudgetBytes int                  `json:"max_budget_bytes"`
}
