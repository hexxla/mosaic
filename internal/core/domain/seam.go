package domain

import "time"

// Seam invariants when backed by HexxlaDB:
//
//   - MarkConflict canonically orders the two-cell endpoints; MarkSupersedes is directional
//     (superseded cell → superseder cell) for context-pack supersession filtering.
//
//   - Empty ResolutionStatus means unresolved; FindSeams can filter to unresolved-only.
//
//   - New seam ids are ULIDs assigned by the engine for MarkConflict and MarkSupersedes;
//     ResolveSeam updates an existing seam by id.

// FindSeamsQuery lists seams incident to cells within hex distance radius of center.
type FindSeamsQuery struct {
	AsOf     *time.Time
	SourceID string
	Center   AxialCoord
	// Radius is the maximum hex ring distance from Center (must be >= 0; services cap aggressively).
	Radius int
	// UnresolvedOnly, when true, keeps seams whose ResolutionStatus is empty.
	UnresolvedOnly bool
}

// SeamHit is one decoded seam for MCP / application use (no packed-key types).
type SeamHit struct {
	Confidence        float64    `json:"confidence,omitzero"`
	CreatedAt         string     `json:"created_at,omitempty"`
	UpdatedAt         string     `json:"updated_at,omitempty"`
	ValidFrom         string     `json:"valid_from,omitempty"`
	ValidTo           string     `json:"valid_to,omitempty"`
	ID                string     `json:"id" jsonschema:"seam ULID"`
	CellA             AxialCoord `json:"cell_a"`
	CellB             AxialCoord `json:"cell_b"`
	SeamType          string     `json:"seam_type"`
	Reason            string     `json:"reason,omitempty"`
	ConfidenceDelta   float64    `json:"confidence_delta,omitempty"`
	DetectedAtRFC3339 string     `json:"detected_at_rfc3339,omitempty"`
	ResolutionStatus  string     `json:"resolution_status,omitempty"`
	ResolutionNote    string     `json:"resolution_note,omitempty"`
	SourceID          string     `json:"source_id,omitempty"`
}

// FindSeamsResponse wraps seam listing (order engine-defined).
type FindSeamsResponse struct {
	Seams []SeamHit `json:"seams"`
}

// PutSeamCommand writes an explicitly identified custom seam without inventing an ID on retry.
type PutSeamCommand struct {
	CreatedAt        *time.Time `json:"created_at,omitempty"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
	ID               string     `json:"id"`
	CellA            AxialCoord `json:"cell_a"`
	CellB            AxialCoord `json:"cell_b"`
	SeamType         string     `json:"seam_type"`
	Reason           string     `json:"reason,omitempty"`
	ConfidenceDelta  float64    `json:"confidence_delta,omitzero"`
	DetectedAt       *time.Time `json:"detected_at,omitempty"`
	ResolutionStatus string     `json:"resolution_status,omitempty"`
	ResolutionNote   string     `json:"resolution_note,omitempty"`
	ValidFrom        *time.Time `json:"valid_from,omitempty"`
	ValidTo          *time.Time `json:"valid_to,omitempty"`
	SourceID         string     `json:"source_id"`
	Confidence       float64    `json:"confidence,omitzero"`
}
