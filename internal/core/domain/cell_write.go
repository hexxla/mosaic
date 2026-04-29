package domain

// CellPutKind selects which HexxlaDB cell template to use before merging Tags.
// Fact is the generic path (arbitrary tags); user_message and assistant_response follow
// conversational_memory-style provenance (session id in SourceID).
type CellPutKind string

const (
	// CellPutKindFact uses a fact-style template with SourceID as provenance source (default).
	CellPutKindFact CellPutKind = "fact"
	// CellPutKindUserMessage uses standard user-message tags; SourceID acts as session identifier.
	CellPutKindUserMessage CellPutKind = "user_message"
	// CellPutKindAssistantResponse uses standard assistant tags; SourceID acts as session identifier.
	CellPutKindAssistantResponse CellPutKind = "assistant_response"
)

// PutCellCommand writes one cell at an axial coordinate (within DB.Update).
type PutCellCommand struct {
	Coord      AxialCoord
	RawContent string
	Tags       []string
	SourceID   string
	Confidence float64
	// Kind selects the record template; empty means [CellPutKindFact].
	Kind CellPutKind
}

// PutEmbeddingCommand stores a vector at coord: supply either embed Text (Ollama) or Vector (raw).
type PutEmbeddingCommand struct {
	Coord  AxialCoord
	Text   string // natural language; embedded via Ollama when non-empty
	Vector []float32
}

// DeleteCellCommand removes the cell at the given axial coordinate (tombstone on MVCC DBs).
type DeleteCellCommand struct {
	Coord AxialCoord
}

// MutationOK confirms a successful write or delete MCP round-trip.
type MutationOK struct {
	OK bool `json:"ok"`
}
