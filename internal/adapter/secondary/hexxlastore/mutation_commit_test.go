package hexxlastore

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hexxla/hexxladb"

	"github.com/hexxla/mosaic/internal/core/domain"
)

func TestClassifyMutationCommit(t *testing.T) {
	cause := errors.New("storage failure")
	durable := classifyMutationCommit(fmt.Errorf("%w: %w: %w", hexxladb.ErrCommitFinalization, hexxladb.ErrCommitDurable, cause))
	if !errors.Is(durable, domain.ErrMutationCommitDurable) || errors.Is(durable, domain.ErrMutationCommitUncertain) ||
		!errors.Is(durable, cause) || !errors.Is(durable, hexxladb.ErrCommitFinalization) {
		t.Fatalf("known-durable error = %v", durable)
	}
	uncertain := classifyMutationCommit(fmt.Errorf("%w: %w", hexxladb.ErrCommitFinalization, cause))
	if !errors.Is(uncertain, domain.ErrMutationCommitUncertain) || errors.Is(uncertain, domain.ErrMutationCommitDurable) ||
		!errors.Is(uncertain, cause) || !errors.Is(uncertain, hexxladb.ErrCommitFinalization) {
		t.Fatalf("uncertain error = %v", uncertain)
	}
	if err := classifyMutationCommit(cause); !errors.Is(err, cause) ||
		errors.Is(err, domain.ErrMutationCommitDurable) || errors.Is(err, domain.ErrMutationCommitUncertain) {
		t.Fatalf("ordinary failure = %v", err)
	}
}
