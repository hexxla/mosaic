package domain

import "errors"

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

// CellPlacementMode selects how Mosaic resolves a coordinate for a new cell.
type CellPlacementMode string

const (
	// CellPlacementExact writes at Coord after applying the explicit overwrite policy.
	CellPlacementExact CellPlacementMode = "exact"
	// CellPlacementNearAnchor selects the first unused coordinate around Coord.
	CellPlacementNearAnchor CellPlacementMode = "near_anchor"
)

// ErrCellCoordinateOccupied means an exact placement targeted a tombstone
// or a live cell without explicitly allowing replacement.
var ErrCellCoordinateOccupied = errors.New("cell coordinate occupied")

// ErrCellContentChanged means an embedding guard no longer matches its cell.
var ErrCellContentChanged = errors.New("cell content changed")

// ErrIdempotencyKeyConflict means the key already identifies a different write.
var ErrIdempotencyKeyConflict = errors.New("idempotency_key already used for different put_cell arguments")

// ErrMutationCommitDurable means the primary commit succeeded and recovery is required.
var ErrMutationCommitDurable = errors.New("commit_outcome=durable: mutation saved; reopen the database to finish recovery")

// ErrMutationCommitUncertain means recovery is required before reconciling the write.
var ErrMutationCommitUncertain = errors.New("commit_outcome=uncertain: reopen the database before reconciling or retrying the mutation")

// PutCellCommand writes one cell at an axial coordinate (within DB.Update).
type PutCellCommand struct {
	Coord      AxialCoord
	RawContent string
	Tags       []string
	SourceID   string
	Confidence float64
	// Kind selects the record template; empty means [CellPutKindFact].
	Kind CellPutKind

	// Placement defaults to [CellPlacementExact]. In near-anchor mode Coord is
	// the semantic anchor and MaxRadius bounds deterministic free-cell search.
	Placement      CellPlacementMode
	MaxRadius      int
	AllowOverwrite bool
	Embed          bool // compute and commit an embedding with this cell when requested
	// IdempotencyKey identifies one near-anchor write across sessions and restarts.
	IdempotencyKey string
}

// PutEmbeddingCommand stores a vector at coord: supply either embed Text (Ollama) or Vector (raw).
type PutEmbeddingCommand struct {
	Coord  AxialCoord
	Text   string // natural language; embedded via Ollama when non-empty
	Vector []float32
	// ExpectedContentSHA256 optionally guards the current RawContent in the
	// same transaction as the embedding write. Empty disables the guard.
	ExpectedContentSHA256 string
}

// DeleteCellCommand removes the cell at the given axial coordinate (tombstone on MVCC DBs).
type DeleteCellCommand struct {
	Coord AxialCoord
}

// MutationOK confirms a successful simple mutation MCP round-trip.
type MutationOK struct {
	OK bool `json:"ok"`
}

// PutCellMutationResult reports the actual coordinate selected for a cell write.
type PutCellMutationResult struct {
	Embedded  bool              `json:"embedded"`
	OK        bool              `json:"ok"`
	Coord     AxialCoord        `json:"coord"`
	Placement CellPlacementMode `json:"placement"`
	Probes    int               `json:"probes"`
	Replaced  bool              `json:"replaced"`
}

// DeleteCellMutationResult is returned by mosaic_hexxla_delete_cell.
type DeleteCellMutationResult struct {
	OK          bool `json:"ok"`
	CellRemoved bool `json:"cell_removed"`
}
