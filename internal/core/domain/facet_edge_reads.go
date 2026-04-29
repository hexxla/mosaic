package domain

// GetFacetResponse is Tx.GetFacet semantics: found=false when the slot has no visible facet.
type GetFacetResponse struct {
	Found bool `json:"found"`
	Q     int  `json:"q"`
	R     int  `json:"r"`

	FacetID        uint8  `json:"facet_id"`
	DerivedContent string `json:"derived_content,omitempty"`

	// LastRotatedUnixNano is engine facet rotation timestamp when present (nanoseconds UTC).
	LastRotatedUnixNano int64 `json:"last_rotated_unix_nano,omitempty"`
	// DerivationHashHex is SHA-256 (hex) of host cell RawContent at write time — empty string when all-zero.
	DerivationHashHex string `json:"derivation_hash_hex,omitempty"`
}

// ListFacetsForCellResponse aggregates visible facets at a cell (Tx.AscendFacetsForCell).
type ListFacetsForCellResponse struct {
	Q      int               `json:"q"`
	R      int               `json:"r"`
	Facets []FacetSlotBullet `json:"facets"`
}

// FacetSlotBullet is one facet row for MCP JSON (slot + payload).
type FacetSlotBullet struct {
	FacetID        uint8  `json:"facet_id"`
	DerivedContent string `json:"derived_content,omitempty"`

	LastRotatedUnixNano int64  `json:"last_rotated_unix_nano,omitempty"`
	DerivationHashHex   string `json:"derivation_hash_hex,omitempty"`
}

// GetEdgeResponse is Tx.GetEdge semantics: found=false when no matching edge exists.
type GetEdgeResponse struct {
	Found bool `json:"found"`

	FromQ int `json:"from_q"`
	FromR int `json:"from_r"`
	ToQ   int `json:"to_q"`
	ToR   int `json:"to_r"`

	RelationType string  `json:"relation_type,omitempty"`
	Weight       float64 `json:"weight,omitempty"`

	SourceID    string  `json:"source_id,omitempty"`
	Confidence  float64 `json:"confidence,omitempty"`
	CreatedAtNs int64   `json:"created_at_unix_nano,omitempty"`
	UpdatedAtNs int64   `json:"updated_at_unix_nano,omitempty"`
}

// EdgeBullet is one outbound edge from a pivot cell for JSON output.
type EdgeBullet struct {
	ToQ          int     `json:"to_q"`
	ToR          int     `json:"to_r"`
	RelationType string  `json:"relation_type"`
	Weight       float64 `json:"weight"`

	SourceID    string  `json:"source_id,omitempty"`
	Confidence  float64 `json:"confidence,omitempty"`
	CreatedAtNs int64   `json:"created_at_unix_nano,omitempty"`
	UpdatedAtNs int64   `json:"updated_at_unix_nano,omitempty"`
}

// ListEdgesFromResponse collects Tx.AscendEdgesFrom edges up to MaxEdgesApplied.
type ListEdgesFromResponse struct {
	FromQ int `json:"from_q"`
	FromR int `json:"from_r"`

	Edges           []EdgeBullet `json:"edges"`
	MaxEdgesApplied int          `json:"max_edges_applied"`
	// Truncated is true when at least one additional edge existed beyond MaxEdgesApplied.
	Truncated bool `json:"truncated"`
}
