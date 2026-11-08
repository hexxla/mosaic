package mcpsrv

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	ratchetadapters "github.com/hexxla/mcp-ratchet/pkg/ratchet/adapters"
	ratchetdomain "github.com/hexxla/mcp-ratchet/pkg/ratchet/domain"
	ratchetservices "github.com/hexxla/mcp-ratchet/pkg/ratchet/services"
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
	rules := []ratchetdomain.Rule{
		{Tool: "protected", Prerequisite: "discover"},
		{Tool: "free-pass"},
		{Tool: "mixed", Prerequisite: "discover"},
		{Tool: "mixed"},
	}
	gate := NewRatchetGate(nil, nil, rules, nil)

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

func newRatchetTestServer(t *testing.T, failDiscovery bool, mutationCalls *atomic.Int32) *httptest.Server {
	t.Helper()

	rules := []ratchetdomain.Rule{
		{Tool: "discover", Expiry: time.Minute},
		{
			Tool:         "mutate",
			Prerequisite: "discover",
			ErrorMessage: "discover first",
			OneTimeUse:   true,
		},
	}
	sessions := ratchetadapters.NewMemorySessionStore()
	service := ratchetservices.NewRatchetService(
		ratchetadapters.NewYAMLConfigLoader(),
		ratchetadapters.NewMemoryTokenStore(),
		sessions,
		ratchetadapters.NewCryptoRandomGenerator(),
		ratchetadapters.NewRealClock(),
	)
	for _, rule := range rules {
		if err := service.RegisterRule(t.Context(), rule); err != nil {
			t.Fatalf("register rule: %v", err)
		}
	}

	srv := NewServer("ratchet-test", "test", nil)
	srv.AddReceivingMiddleware(NewRatchetGate(service, sessions, rules, nil).Middleware())
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

func TestRatchetGateRejectsOneTimeAlternativesOnLegacyDependency(t *testing.T) {
	rules := []ratchetdomain.Rule{
		{Tool: "mutate", Prerequisite: "discover-a", OneTimeUse: true},
		{Tool: "mutate", Prerequisite: "discover-b", OneTimeUse: true},
	}
	gate := NewRatchetGate(nil, nil, rules, nil)
	if err := gate.ValidateProtectedTools([]string{"mutate"}); err == nil {
		t.Fatal("accepted an unsupported one-time alternative policy")
	}
}

func TestRatchetGateDelegatesOneTimeAlternatives(t *testing.T) {
	rules := []ratchetdomain.Rule{
		{Tool: "discover_b"},
		{Tool: "mutate", Prerequisite: "discover_a", OneTimeUse: true},
		{Tool: "mutate", Prerequisite: "discover_b", OneTimeUse: true},
	}
	sessions := ratchetadapters.NewMemorySessionStore()
	service := ratchetservices.NewRatchetService(ratchetadapters.NewYAMLConfigLoader(), ratchetadapters.NewMemoryTokenStore(), sessions, ratchetadapters.NewCryptoRandomGenerator(), ratchetadapters.NewRealClock())
	if _, supported := service.(workflowExecutor); !supported {
		t.Skip("requires a Ratchet module with workflow execution")
	}
	for _, rule := range rules {
		if err := service.RegisterRule(t.Context(), rule); err != nil {
			t.Fatal(err)
		}
	}
	gate := NewRatchetGate(service, sessions, rules, nil)
	if err := gate.ValidateProtectedTools([]string{"mutate"}); err != nil {
		t.Fatal(err)
	}
	var calls int
	for i, tool := range []ratchetdomain.ToolName{"discover_b", "mutate", "mutate"} {
		result, err := gate.execute(t.Context(), "test-session", tool, func() (mcp.Result, error) {
			if tool == "mutate" {
				calls++
			}
			return new(mcp.CallToolResult), nil
		})
		if err != nil {
			t.Fatal(err)
		}
		callResult, ok := result.(*mcp.CallToolResult)
		if !ok || callResult.IsError != (i == 2) {
			t.Fatalf("call %d returned unexpected result: %v", i, result)
		}
	}
	if calls != 1 {
		t.Fatalf("one-time prerequisite authorized %d mutations", calls)
	}
}
