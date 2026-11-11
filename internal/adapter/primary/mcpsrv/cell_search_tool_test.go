package mcpsrv

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hexxla/mosaic/internal/core/domain"
)

type searchCapture struct {
	search *domain.CellSearchCommand
	coord  domain.AxialCoord
}

func (c *searchCapture) SearchCells(_ context.Context, q *domain.CellSearchCommand) (domain.CellHitsResponse, error) {
	c.search = q
	return domain.CellHitsResponse{Hits: []domain.CellHit{}, Mode: q.Mode, ScanLimitReached: true}, nil
}
func (c *searchCapture) GetCell(_ context.Context, p domain.AxialCoord) (domain.CellLookupResponse, error) {
	c.coord = p
	return domain.CellLookupResponse{}, nil
}

func TestCanonicalSearchAndExactLookupMCP(t *testing.T) {
	capture := &searchCapture{}
	srv := NewServer("search-test", "test", nil)
	if err := RegisterCellSearchTool(srv, capture, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := RegisterGetCellTool(srv, capture, nil, nil); err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(StreamableHTTPHandler(srv, nil))
	defer httpServer.Close()
	client, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).Connect(t.Context(), &mcp.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()
	args := map[string]any{"query": "semantic cue", "mode": "hybrid", "lexical_query": "literal", "require_tags": []string{"topic"}, "exclude_tags": []string{"skip"}, "center_q": 100, "center_r": -4, "radius": 2, "after_rfc3339": "2026-01-01T00:00:00Z", "sort_by": "confidence", "max_scan_rows": 7}
	result, err := client.CallTool(t.Context(), &mcp.CallToolParams{Name: "mosaic_hexxla_search_cells", Arguments: args})
	if err != nil || result.IsError {
		t.Fatalf("search=%+v %v", result, err)
	}
	q := capture.search
	if q == nil || q.Mode != domain.SearchHybrid || q.Query != "semantic cue" || q.LexicalQuery != "literal" || q.Center.Q != 100 || q.Radius != 2 || q.MaxScanRows != 7 || q.After == nil || q.SortBy != domain.CellQuerySortConfidence || len(q.ExcludeTags) != 1 {
		t.Fatalf("mapped command=%+v", q)
	}
	var out domain.CellHitsResponse
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &out); err != nil {
		t.Fatal(err)
	}
	if !out.ScanLimitReached || out.Hits == nil {
		t.Fatalf("lost partial/empty contract: %+v", out)
	}
	result, err = client.CallTool(t.Context(), &mcp.CallToolParams{Name: "mosaic_hexxla_get_cell", Arguments: map[string]any{"q": 100, "r": -4}})
	if err != nil || result.IsError {
		t.Fatalf("get=%+v %v", result, err)
	}
	if capture.coord != (domain.AxialCoord{Q: 100, R: -4}) {
		t.Fatalf("lookup coord=%+v", capture.coord)
	}
	var lookup domain.CellLookupResponse
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &lookup); err != nil {
		t.Fatal(err)
	}
	if lookup.Found || lookup.Cell != nil {
		t.Fatalf("absence=%+v", lookup)
	}
	tools, err := client.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 2 {
		t.Fatalf("unexpected discovery surface: %d", len(tools.Tools))
	}
	for _, tool := range tools.Tools {
		if tool.Name == "mosaic_hexxla_query_cells" || tool.Name == "mosaic_hexxla_search_embedding" {
			t.Fatal("superseded discovery tool remains")
		}
	}
}
