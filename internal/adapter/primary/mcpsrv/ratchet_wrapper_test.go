package mcpsrv

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hexxla/mcp-ratchet/pkg/ratchet"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ratchetTestOutput struct {
	OK bool `json:"ok"`
}

func TestRatchetGateIsolatesSessionsAndConsumesOneTimeToken(t *testing.T) {
	var mutationCalls atomic.Int32
	server := newRatchetTestServer(t, false, &mutationCalls)
	clientA := connectRatchetTestClient(t, server.URL)
	clientB := connectRatchetTestClient(t, server.URL)

	assertToolError(t, callRatchetTestTool(t, clientA, "mutate"))
	assertToolSuccess(t, callRatchetTestTool(t, clientA, "discover"))
	assertToolError(t, callRatchetTestTool(t, clientB, "mutate"))
	assertToolSuccess(t, callRatchetTestTool(t, clientA, "mutate"))
	assertToolError(t, callRatchetTestTool(t, clientA, "mutate"))

	if got := mutationCalls.Load(); got != 1 {
		t.Fatalf("mutation handler calls = %d, want 1", got)
	}
}

func TestRatchetGateDoesNotAuthorizeFailedTool(t *testing.T) {
	var mutationCalls atomic.Int32
	server := newRatchetTestServer(t, true, &mutationCalls)
	client := connectRatchetTestClient(t, server.URL)

	assertToolError(t, callRatchetTestTool(t, client, "discover"))
	assertToolError(t, callRatchetTestTool(t, client, "mutate"))

	if got := mutationCalls.Load(); got != 0 {
		t.Fatalf("mutation handler calls = %d, want 0", got)
	}
}

func TestRatchetGateValidatesProtectedToolCoverage(t *testing.T) {
	rules := []ratchet.Rule{
		{Tool: "protected", AfterAny: []string{"discover"}},
		{Tool: "free-pass"},
		{Tool: "mixed", AfterAny: []string{"discover"}},
		{Tool: "mixed"},
	}
	gate := NewRatchetGate(nil, ratchet.Policy{Rules: rules})

	if err := gate.ValidateProtectedTools([]string{"protected"}); err != nil {
		t.Fatalf("validate protected tool: %v", err)
	}
	err := gate.ValidateProtectedTools([]string{"protected", "missing", "free-pass", "mixed"})
	if err == nil {
		t.Fatal("validate coverage unexpectedly succeeded")
	}
	for _, name := range []string{"free-pass", "missing", "mixed"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("coverage error %q does not name %q", err, name)
		}
	}
}

func TestRatchetGateRejectsUnknownPrerequisite(t *testing.T) {
	gate := NewRatchetGate(nil, ratchet.Policy{Rules: []ratchet.Rule{
		{Tool: "mosaic_hexxla_put_cell", AfterAny: []string{"typo"}},
	}})
	if err := gate.ValidateToolNames(); err == nil || !strings.Contains(err.Error(), "typo") {
		t.Fatalf("policy validation error = %v", err)
	}
}

func newRatchetTestServer(t *testing.T, failDiscovery bool, mutationCalls *atomic.Int32) *httptest.Server {
	t.Helper()

	rules := []ratchet.Rule{
		{Tool: "discover", TTL: time.Minute},
		{
			Tool:     "mutate",
			AfterAny: []string{"discover"},
			Message:  "discover first",
			Consume:  true,
		},
	}
	policy := ratchet.Policy{Rules: rules}
	service, err := ratchet.New(policy, ratchet.Options{})
	if err != nil {
		t.Fatal(err)
	}

	srv := NewServer("ratchet-test", "test", nil)
	srv.AddReceivingMiddleware(NewRatchetGate(service, policy).Middleware())
	mcp.AddTool(srv, &mcp.Tool{Name: "discover"}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, ratchetTestOutput, error) {
		if failDiscovery {
			return nil, ratchetTestOutput{}, errors.New("discovery failed")
		}
		return nil, ratchetTestOutput{OK: true}, nil
	})
	mcp.AddTool(srv, &mcp.Tool{Name: "mutate"}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, ratchetTestOutput, error) {
		mutationCalls.Add(1)
		return nil, ratchetTestOutput{OK: true}, nil
	})

	testServer := httptest.NewServer(StreamableHTTPHandler(srv, nil))
	t.Cleanup(testServer.Close)
	return testServer
}

func connectRatchetTestClient(t *testing.T, endpoint string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "ratchet-test-client", Version: "test"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{
		Endpoint:             endpoint,
		DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatalf("connect MCP client: %v", err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close MCP session: %v", err)
		}
	})
	return session
}

func callRatchetTestTool(t *testing.T, session *mcp.ClientSession, name string) *mcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name})
	if err != nil {
		t.Fatalf("call tool %q: %v", name, err)
	}
	return result
}

func assertToolSuccess(t *testing.T, result *mcp.CallToolResult) {
	t.Helper()
	if result.IsError {
		t.Fatalf("tool returned error: %+v", result.Content)
	}
}

func assertToolError(t *testing.T, result *mcp.CallToolResult) {
	t.Helper()
	if !result.IsError {
		t.Fatalf("tool unexpectedly succeeded: %+v", result.Content)
	}
}

func TestRatchetGateDelegatesOneTimeAlternatives(t *testing.T) {
	policy := ratchet.Policy{Rules: []ratchet.Rule{{Tool: "mutate", AfterAny: []string{"discover_a", "discover_b"}, Consume: true}}}
	engine, err := ratchet.New(policy, ratchet.Options{})
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer("alternatives", "test", nil)
	srv.AddReceivingMiddleware(NewRatchetGate(engine, policy).Middleware())
	var calls atomic.Int32
	for _, name := range []string{"discover_a", "discover_b", "mutate"} {
		mcp.AddTool(srv, &mcp.Tool{Name: name}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, struct{}, error) {
			if name == "mutate" {
				calls.Add(1)
			}
			return nil, struct{}{}, nil
		})
	}
	server := httptest.NewServer(StreamableHTTPHandler(srv, nil))
	t.Cleanup(server.Close)
	client := connectRatchetTestClient(t, server.URL)
	assertToolSuccess(t, callRatchetTestTool(t, client, "discover_b"))
	assertToolSuccess(t, callRatchetTestTool(t, client, "mutate"))
	assertToolError(t, callRatchetTestTool(t, client, "mutate"))
	if calls.Load() != 1 {
		t.Fatalf("one completion allowed %d mutations", calls.Load())
	}
}
