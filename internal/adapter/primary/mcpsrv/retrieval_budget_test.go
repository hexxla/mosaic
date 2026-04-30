package mcpsrv

import (
	"errors"
	"testing"

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
	out, err := RunBudgetedRead(rb, nil, func() (int, error) { return 42, nil })
	if err != nil || out != 42 {
		t.Fatalf("got %v err=%v", out, err)
	}
}

func TestRunBudgetedRead_fnError(t *testing.T) {
	t.Parallel()
	rb := NewRetrievalBudgetTracker(config.RetrievalBudgetConfig{})
	want := errors.New("boom")
	_, err := RunBudgetedRead(rb, nil, func() (int, error) { return 0, want })
	if !errors.Is(err, want) {
		t.Fatalf("got %v want %v", err, want)
	}
}

func TestRunBudgetedRead_singleResponseExceedsSessionBudget(t *testing.T) {
	t.Parallel()
	rb := NewRetrievalBudgetTracker(config.RetrievalBudgetConfig{SessionApproxTokenBudget: 2, BytesPerApproxToken: 1})
	_, err := RunBudgetedRead(rb, nil, func() (string, error) { return "hello", nil })
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
	_, err := RunBudgetedRead(rb, nil, func() (map[string]string, error) {
		return map[string]string{"k": "v"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if rb.Status(nil).ApproxTokensUsed <= 0 {
		t.Fatal("expected metering without enforcement")
	}
}
