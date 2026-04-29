package domain

// EmbeddingSearchQuery is a natural-language query for ANN search over stored embeddings.
type EmbeddingSearchQuery struct {
	Text string

	// MaxResults caps hits (default applied in service if zero).
	MaxResults int

	// MinScore filters by similarity when > 0 (engine-specific scale; see HexxlaDB docs).
	MinScore float64
}

// AxialCoord is a lattice axial coordinate (q, r); cube s = -q - r.
type AxialCoord struct {
	Q int `json:"q" jsonschema:"axial q"`
	R int `json:"r" jsonschema:"axial r"`
}

// EmbeddingMatch is one hit from vector similarity plus visible cell payload fields.
type EmbeddingMatch struct {
	Coord      AxialCoord `json:"coord" jsonschema:"cell coordinate"`
	Score      float64    `json:"score" jsonschema:"similarity score from engine"`
	RawContent string     `json:"raw_content,omitempty" jsonschema:"decoded cell text"`
	Tags       []string   `json:"tags,omitempty" jsonschema:"cell tags"`

	SourceID   string  `json:"source_id,omitempty"`
	Confidence float64 `json:"confidence"`

	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	ValidFrom string `json:"valid_from,omitempty"`
	ValidTo   string `json:"valid_to,omitempty"`
}

// EmbeddingSearchResponse is the MCP-facing result for semantic search.
type EmbeddingSearchResponse struct {
	Query         string           `json:"query" jsonschema:"original query text"`
	Matches       []EmbeddingMatch `json:"matches" jsonschema:"ranked hits"`
	RetrievalHint string           `json:"retrieval_hint,omitempty" jsonschema:"when to chain mosaic_hexxla_load_context_pack"`
}
