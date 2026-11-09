package domain

import "time"

// CellQuerySort selects ordering for QueryCells (maps to engine sort modes).
type CellQuerySort string

// Sort modes for [CellSearchCommand.SortBy].
const (
	CellQuerySortScore       CellQuerySort = "score"
	CellQuerySortConfidence  CellQuerySort = "confidence"
	CellQuerySortRecency     CellQuerySort = "recency"
	CellQuerySortCoord       CellQuerySort = "coord"
	CellQuerySortUnspecified CellQuerySort = ""
)

// SearchMode selects the retrieval score and execution path.
type SearchMode string

// Supported search modes preserve their native scoring semantics.
const (
	SearchLexical  SearchMode = "lexical"
	SearchSemantic SearchMode = "semantic"
	SearchHybrid   SearchMode = "hybrid"
)

// AxialCoord is a lattice axial coordinate; cube s = -q-r.
type AxialCoord struct {
	Q int `json:"q" jsonschema:"axial q"`
	R int `json:"r" jsonschema:"axial r"`
}

// CellSearchCommand provides one bounded retrieval contract. Hybrid Query is
// embedded and LexicalQuery optionally applies an independent literal filter.
// Semantic mode retains pure vector scoring; structured filters use hybrid mode.
type CellSearchCommand struct {
	Query         string
	Mode          SearchMode
	LexicalQuery  string
	RequireTags   []string
	AnyTags       []string
	ExcludeTags   []string
	SourceID      string
	MinConfidence float64
	MaxConfidence float64
	After         *time.Time
	Before        *time.Time
	Center        *AxialCoord
	Radius        int
	MaxResults    int
	MaxScanRows   int
	SortBy        CellQuerySort
	Explain       bool
	MinScore      float64
}

// CellLookupResponse distinguishes missing cells from empty cell content.
type CellLookupResponse struct {
	Found bool     `json:"found"`
	Cell  *CellHit `json:"cell,omitempty"`
}

// CellHit is one ranked cell from QueryCells or SearchCells (MCP-safe projection).
type CellHit struct {
	Coord       AxialCoord `json:"coord"`
	RawContent  string     `json:"raw_content,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	Score       float64    `json:"score"`
	Explanation string     `json:"explanation,omitempty"`
	SourceID    string     `json:"source_id,omitempty"`
	Confidence  float64    `json:"confidence"`

	// Provenance and validity (RFC3339Nano UTC) from cell wire; omitted when unknown or zero on wire.
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	ValidFrom string `json:"valid_from,omitempty"`
	ValidTo   string `json:"valid_to,omitempty"`
}

// CellHitsResponse is the MCP JSON body for query/search cell tools.
type CellHitsResponse struct {
	Mode                           SearchMode `json:"mode" jsonschema:"score semantics: lexical, pure-vector semantic, or confidence-aware hybrid"`
	Hits                           []CellHit  `json:"hits" jsonschema:"ranked cell rows"`
	RetrievalHint                  string     `json:"retrieval_hint,omitempty" jsonschema:"when to chain mosaic_hexxla_load_context_pack"`
	ScanLimitReached               bool       `json:"scan_limit_reached,omitempty" jsonschema:"results are partial because the scan row budget was exhausted"`
	EmbeddingPath                  string     `json:"embedding_path,omitempty" jsonschema:"vector execution path: hnsw, flat, or none"`
	EmbeddingCandidates            int        `json:"embedding_candidates,omitempty" jsonschema:"distinct vector candidates examined"`
	EmbeddingCandidateLimitReached bool       `json:"embedding_candidate_limit_reached,omitempty" jsonschema:"vector widening stopped at the candidate budget before filling the requested result count"`
}
