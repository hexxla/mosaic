package hexxlastore

import (
	"bytes"

	"github.com/hexxla/hexxladb"
)

// HasRetainedChangefeed reports whether the primary still has state that must
// be opened with its matching sidecar. The caller holds the database file lock.
func HasRetainedChangefeed(db *hexxladb.DB) (bool, error) {
	consumers, err := db.ListChangelogConsumers()
	if err != nil || len(consumers) > 0 {
		return len(consumers) > 0, err
	}

	// The pinned HexxlaDB API exposes consumer cursors but has no history
	// status accessor. Check presence without decoding its private values.
	var retained bool
	err = db.View(func(tx *hexxladb.Tx) error {
		_, found, err := tx.Get([]byte("__meta/changelog-projection-checkpoint"))
		if err != nil || found {
			retained = found
			return err
		}
		prefix := []byte("__meta/changelog-outbox/")
		return tx.AscendRange(prefix, []byte("__meta/changelog-outbox0"), func(key, _ []byte) bool {
			retained = bytes.HasPrefix(key, prefix)
			return false
		})
	})
	return retained, err
}
