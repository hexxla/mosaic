package mcpsrv

import (
	"testing"
)

func TestValidateContextPackBudgetInput(t *testing.T) {
	t.Parallel()
	t.Run("omit_alone_ok", func(t *testing.T) {
		t.Parallel()
		if err := validateContextPackBudgetInput(&contextPackInput{OmitBudget: true}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("omit_with_max_tokens", func(t *testing.T) {
		t.Parallel()
		if err := validateContextPackBudgetInput(&contextPackInput{OmitBudget: true, MaxTokens: 4096}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("token_and_max_tokens", func(t *testing.T) {
		t.Parallel()
		if err := validateContextPackBudgetInput(&contextPackInput{BudgetTokensApprox: 100, MaxTokens: 1}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("dual_explicit_same", func(t *testing.T) {
		t.Parallel()
		if err := validateContextPackBudgetInput(&contextPackInput{MaxBudgetBytes: 5000, MaxTokens: 5000}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("dual_explicit_conflict", func(t *testing.T) {
		t.Parallel()
		if err := validateContextPackBudgetInput(&contextPackInput{MaxBudgetBytes: 5000, MaxTokens: 4000}); err == nil {
			t.Fatal("expected error")
		}
	})
}
