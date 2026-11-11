package hexxlastore

import (
	"errors"
	"fmt"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

func updateMutation(db *hexxladb.DB, apply func(*hexxladb.Tx) error) error {
	return classifyMutationCommit(db.Update(apply))
}

func classifyMutationCommit(err error) error {
	switch {
	case errors.Is(err, hexxladb.ErrCommitDurable):
		return fmt.Errorf("%w: %w", domain.ErrMutationCommitDurable, err)
	case errors.Is(err, hexxladb.ErrCommitFinalization):
		return fmt.Errorf("%w: %w", domain.ErrMutationCommitUncertain, err)
	default:
		return err
	}
}
