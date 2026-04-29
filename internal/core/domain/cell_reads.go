package domain

import "time"

// CellQuerySort selects ordering for QueryCells (maps to engine sort modes).
type CellQuerySort string

// Sort modes for [CellQueryCommand.SortBy].
const (
	CellQuerySortScore       CellQuerySort = "score"
	CellQuerySortConfidence  CellQuerySort = "confidence"
	CellQuerySortRecency     CellQuerySort = "recency"
	CellQuerySortCoord       CellQuerySort = "coord"
	CellQuerySortUnspecified CellQuerySort = ""
)

// CellQueryCommand is a structured predicate for indexed cell reads (Hexxla Tx.QueryCells).
type CellQueryCommand struct {
	Query string

	RequireTags []string
	AnyTags     []string
	ExcludeTags []string

	SourceID string

	MinConfidence float64
	MaxConfidence float64

	After  *time.Time
	Before *time.Time

	Center *AxialCoord
	Radius int

	MaxResults  int
	MaxScanRows int

	SortBy  CellQuerySort
	Explain bool

	// EmbedQueryText enables hybrid retrieval: embed this natural-language string via Ollama
	// then pass the vector into Hexxla QueryCells (ANN-accelerated seed selection + predicates).
	// Empty disables hybrid mode. Requires a database opened with non-zero embedding dimension.
	EmbedQueryText string `json:"embed_query_text,omitempty"`
}

// CellSearchCommand is lexical relevance search over cells (Hexxla Tx.SearchCells).
type CellSearchCommand struct {
	Query string

	RequireTags []string
	AnyTags     []string

	MinConfidence float64
	MaxConfidence float64

	SourceID string

	Center *AxialCoord
	Radius int

	MaxResults    int
	MaxScanRadius int

	// EmbedQueryText enables hybrid retrieval: embed then SearchCells with ANN-accelerated
	// candidate selection combined with lexical score. Empty disables hybrid mode.
	EmbedQueryText string `json:"embed_query_text,omitempty"`
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
	Hits          []CellHit `json:"hits" jsonschema:"ranked cell rows"`
	RetrievalHint string    `json:"retrieval_hint,omitempty" jsonschema:"when to chain mosaic_hexxla_load_context_pack"`
}
