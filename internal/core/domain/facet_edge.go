package domain

const (
	// MaxFacetID is the inclusive upper facet slot index accepted by HexxlaDB encoders (0..5).
	MaxFacetID = 5
)

// PutFacetCommand writes derived facet content for one cell facet slot ([0..MaxFacetID]).
// PutFacet ignores derivation-hash coupling; use engine UpdateFacet workflows if you need
// content-hash guarding (not exposed via Mosaic MCP currently).
type PutFacetCommand struct {
	Coord          AxialCoord
	FacetID        uint8
	DerivedContent string
}

// LinkCellsCommand creates or replaces an edge Key(from→to,type) via the spec helper Tx.LinkCells.
type LinkCellsCommand struct {
	From, To AxialCoord
	// RelationType is a non-empty relationship label stored in the edge key.
	RelationType string
	Weight       float64
	SourceID     string
	Confidence   float64
}
