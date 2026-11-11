package domain_test

import (
	"math"
	"testing"

	"github.com/hexxla/mosaic/internal/core/domain"
)

func TestApproximateByteBudgetLargeInputsRemainCapped(t *testing.T) {
	t.Parallel()
	previous := 0
	for _, tokens := range []int{1, 100, 25000, math.MaxInt / 4, math.MaxInt / 2, math.MaxInt} {
		got, err := domain.ApproximateByteBudgetFromTokens(tokens, 4, 64, 100000)
		if err != nil {
			t.Fatal(err)
		}
		if got < previous || got < 64 || got > 100000 {
			t.Fatalf("tokens=%d budget=%d previous=%d", tokens, got, previous)
		}
		if tokens >= 25000 && got != 100000 {
			t.Fatalf("tokens=%d budget=%d, want maximum", tokens, got)
		}
		previous = got
	}
}

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
