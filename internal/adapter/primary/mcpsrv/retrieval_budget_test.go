package mcpsrv

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/config"
)

func TestApproximateTokensFromJSONBytes(t *testing.T) {
	t.Parallel()
	if approximateTokensFromJSONBytes(0, 4) != 0 {
		t.Fatal("empty payload")
	}
	if approximateTokensFromJSONBytes(8, 4) != 2 {
		t.Fatalf("got %d want 2", approximateTokensFromJSONBytes(8, 4))
	}
	if approximateTokensFromJSONBytes(1, 4) != 1 {
		t.Fatalf("got %d want 1", approximateTokensFromJSONBytes(1, 4))
	}
}

func TestNewRetrievalBudgetTracker_disabled(t *testing.T) {
	t.Parallel()
	rb := NewRetrievalBudgetTracker(config.RetrievalBudgetConfig{})
	if rb.Enabled() {
		t.Fatal("expected disabled when budget is 0")
	}
}

func TestRunBudgetedRead_whenBudgetDisabled(t *testing.T) {
	t.Parallel()
	rb := NewRetrievalBudgetTracker(config.RetrievalBudgetConfig{})
	out, err := rb.RunBudgetedRead(nil, func() (int, error) { return 42, nil })
	if err != nil || out != 42 {
		t.Fatalf("got %v err=%v", out, err)
	}
}

func TestRunBudgetedRead_nilTracker(t *testing.T) {
	t.Parallel()
	var tracker *RetrievalBudgetTracker
	out, err := tracker.RunBudgetedRead(nil, func() (int, error) { return 42, nil })
	if err != nil || out != 42 {
		t.Fatalf("got %v err=%v", out, err)
	}
}

func TestRunBudgetedRead_fnError(t *testing.T) {
	t.Parallel()
	rb := NewRetrievalBudgetTracker(config.RetrievalBudgetConfig{})
	want := errors.New("boom")
	_, err := rb.RunBudgetedRead(nil, func() (int, error) { return 0, want })
	if !errors.Is(err, want) {
		t.Fatalf("got %v want %v", err, want)
	}
}

func TestRunBudgetedRead_singleResponseExceedsSessionBudget(t *testing.T) {
	t.Parallel()
	rb := NewRetrievalBudgetTracker(config.RetrievalBudgetConfig{SessionApproxTokenBudget: 2, BytesPerApproxToken: 1})
	_, err := rb.RunBudgetedRead(nil, func() (string, error) { return "hello", nil })
	if err == nil {
		t.Fatal("expected error when one response JSON exceeds session budget")
	}
}

func TestMetering_accumulatesWhenBudgetZero(t *testing.T) {
	t.Parallel()
	rb := NewRetrievalBudgetTracker(config.RetrievalBudgetConfig{})
	if st := rb.Status(nil); st.ApproxTokensUsed != 0 || !st.MeteringEnabled {
		t.Fatalf("initial status: %+v", st)
	}
	_, err := rb.RunBudgetedRead(nil, func() (map[string]string, error) {
		return map[string]string{"k": "v"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if rb.Status(nil).ApproxTokensUsed <= 0 {
		t.Fatal("expected metering without enforcement")
	}
}

func TestRetrievalBudgetReleasesClosedSession(t *testing.T) {
	t.Parallel()
	tracker := NewRetrievalBudgetTracker(config.RetrievalBudgetConfig{SessionApproxTokenBudget: 100})
	server := mcp.NewServer(&mcp.Implementation{Name: "budget-test", Version: "test"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "read"}, func(_ context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, ratchetTestOutput, error) {
		out := ratchetTestOutput{OK: true}
		return nil, out, tracker.RecordStructuredOutput(req, out)
	})
	httpServer := httptest.NewServer(StreamableHTTPHandler(server, nil))
	t.Cleanup(httpServer.Close)
	session := connectRatchetTestClient(t, httpServer.URL)
	assertToolSuccess(t, callRatchetTestTool(t, session, "read"))
	count := func() int {
		tracker.mu.Lock()
		defer tracker.mu.Unlock()
		return len(tracker.used)
	}
	if got := count(); got != 1 {
		t.Fatalf("tracked sessions = %d, want 1", got)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for count() != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := count(); got != 0 {
		t.Fatalf("tracked sessions after close = %d, want 0", got)
	}
}
