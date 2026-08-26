package mcpsrv

import (
	"testing"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

func TestPutCellCommand_maps_placement_inputs(t *testing.T) {
	t.Parallel()
	in := putCellInput{
		Q:              3,
		R:              -2,
		RawContent:     "cell",
		Tags:           []string{"fact"},
		SourceID:       "source",
		Confidence:     0.9,
		Placement:      " near_anchor ",
		MaxRadius:      12,
		AllowOverwrite: false,
	}
	cmd := putCellCommand(in, domain.CellPutKindFact)
	if cmd.Coord != (domain.AxialCoord{Q: 3, R: -2}) || cmd.Placement != domain.CellPlacementNearAnchor || cmd.MaxRadius != 12 {
		t.Fatalf("mapped command: %+v", cmd)
	}
	if len(cmd.Tags) != 1 || cmd.Tags[0] != "fact" {
		t.Fatalf("mapped tags: %+v", cmd.Tags)
	}
	in.Tags[0] = "mutated"
	if cmd.Tags[0] != "fact" {
		t.Fatal("command tags alias MCP input")
	}
}
