package domain_test

import (
	"testing"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

func TestApproximateByteBudgetFromTokens(t *testing.T) {
	t.Parallel()
	const minClamp, maxClamp = 64, 100000

	t.Run("positive_tokens_clamps_floor", func(t *testing.T) {
		t.Parallel()
		// ceil(10*4)=40 → clamp min 64
		got, err := domain.ApproximateByteBudgetFromTokens(10, domain.DefaultApproxBytesPerToken, minClamp, maxClamp)
		if err != nil {
			t.Fatal(err)
		}
		if got != minClamp {
			t.Fatalf("got %d want %d", got, minClamp)
		}
	})

	t.Run("bytes_per_approx_token_validation", func(t *testing.T) {
		t.Parallel()
		_, err := domain.ApproximateByteBudgetFromTokens(100, 1, minClamp, maxClamp)
		if err == nil {
			t.Fatal("expected error below MinApproxBytesPerToken")
		}
	})

	t.Run("zero_tokens", func(t *testing.T) {
		t.Parallel()
		_, err := domain.ApproximateByteBudgetFromTokens(0, 4, minClamp, maxClamp)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
