package mcpsrv

import (
	"context"
	"encoding/json/v2"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hexxla/mosaic/internal/core/domain"
)

func TestContextPackBudgetInput(t *testing.T) {
	if err := validateContextPackBudgetInput(nil); err == nil {
		t.Fatal("nil accepted")
	}
	for _, budget := range []int{0, 64, 4096, 100000} {
		if err := validateContextPackBudgetInput(&contextPackInput{MaxBudgetBytes: budget}); err != nil {
			t.Fatal(err)
		}
	}
	for _, budget := range []int{-1, 63, 100001} {
		if err := validateContextPackBudgetInput(&contextPackInput{MaxBudgetBytes: budget}); err == nil {
			t.Fatalf("invalid budget %d accepted", budget)
		}
	}
}

type contextSchemaService struct{ calls int }

func (s *contextSchemaService) LoadFromSeeds(_ context.Context, cmd *domain.LoadContextPackCommand) (domain.ContextPackResponse, error) {
	s.calls++
	return domain.ContextPackResponse{Cells: []domain.ContextPackCell{}, MaxBudgetBytes: cmd.MaxBudgetBytes}, nil
}

func TestContextPackCanonicalMCPBudgetSchema(t *testing.T) {
	service := new(contextSchemaService)
	srv := NewServer("context-schema", "test", nil)
	if err := RegisterContextPackTool(srv, service, nil, nil); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(StreamableHTTPHandler(srv, nil))
	t.Cleanup(server.Close)
	client := connectRatchetTestClient(t, server.URL)
	tools, err := client.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	input, err := json.Marshal(tools.Tools[0].InputSchema)
	if err != nil {
		t.Fatal(err)
	}
	output, err := json.Marshal(tools.Tools[0].OutputSchema)
	if err != nil {
		t.Fatal(err)
	}
	for _, legacy := range []string{"max_tokens", "omit_budget", "budget_tokens_approx", "bytes_per_approx_token"} {
		if strings.Contains(string(input), `"`+legacy+`"`) {
			t.Fatalf("legacy input exposed: %s", input)
		}
		result, err := client.CallTool(t.Context(), &mcp.CallToolParams{Name: "mosaic_hexxla_load_context_pack", Arguments: map[string]any{"seeds": []map[string]int{{"q": 0, "r": 0}}, legacy: 1}})
		if err != nil || !result.IsError {
			t.Fatalf("legacy %s was not rejected: %+v %v", legacy, result, err)
		}
	}
	if !strings.Contains(string(input), `"max_budget_bytes"`) {
		t.Fatalf("canonical byte budget missing: %s", input)
	}
	for _, legacy := range []string{"total_tokens", "max_tokens_budget"} {
		if strings.Contains(string(output), `"`+legacy+`"`) {
			t.Fatalf("legacy output exposed: %s", output)
		}
	}
	if service.calls != 0 {
		t.Fatalf("invalid legacy requests reached service %d times", service.calls)
	}
	result, err := client.CallTool(t.Context(), &mcp.CallToolParams{Name: "mosaic_hexxla_load_context_pack", Arguments: map[string]any{"seeds": []map[string]int{{"q": 0, "r": 0}}}})
	if err != nil || result.IsError || service.calls != 1 {
		t.Fatalf("optional canonical budget rejected: %+v %v", result, err)
	}
}
