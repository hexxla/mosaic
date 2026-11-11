package mcpsrv

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hexxla/mosaic/internal/core/domain"
	"github.com/hexxla/mosaic/internal/core/ports/primary"
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
	cmd := putCellCommand(&in, domain.CellPutKindFact)
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

type embeddingCommandCapture struct {
	primary.CellMutation
	commands chan *domain.PutEmbeddingCommand
}

func (c embeddingCommandCapture) PutEmbedding(_ context.Context, cmd *domain.PutEmbeddingCommand) error {
	c.commands <- cmd
	return nil
}

func TestPutEmbeddingAcceptsOptionalContentGuard(t *testing.T) {
	t.Parallel()
	capture := embeddingCommandCapture{commands: make(chan *domain.PutEmbeddingCommand, 1)}
	srv := NewServer("embedding-test", "test", nil)
	if err := RegisterPutEmbeddingTool(srv, capture, nil); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(StreamableHTTPHandler(srv, nil))
	t.Cleanup(server.Close)
	client := connectRatchetTestClient(t, server.URL)
	for _, digest := range []string{"", strings.Repeat("a", 64)} {
		args := map[string]any{"q": 1, "r": 2, "vector": []float32{1, 2}}
		if digest != "" {
			args["expected_content_sha256"] = digest
		}
		result, err := client.CallTool(t.Context(), &mcp.CallToolParams{Name: "mosaic_hexxla_put_embedding", Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		assertToolSuccess(t, result)
		if cmd := <-capture.commands; cmd.ExpectedContentSHA256 != digest {
			t.Fatalf("content guard = %q, want %q", cmd.ExpectedContentSHA256, digest)
		}
	}
}
