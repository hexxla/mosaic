package hexxlastore

import (
	"time"

	"github.com/hexxla/hexxladb"
)

// nanoWallToRFC3339 converts Hexxla wire timestamps (unix nanoseconds since epoch, UTC semantics on write).
func nanoWallToRFC3339(ns int64) string {
	if ns <= 0 {
		return ""
	}
	return time.Unix(0, ns).UTC().Format(time.RFC3339Nano)
}

// nanoWallPtrToRFC3339 converts optional *int64 nanosecond wire fields (validity halves).
func nanoWallPtrToRFC3339(p *int64) string {
	if p == nil {
		return ""
	}
	return nanoWallToRFC3339(*p)
}

// timingsFromCellView maps provenance and validity onto domain cell projections.
func timingsFromCellView(v *hexxladb.CellView) (createdAt, updatedAt, validFrom, validTo string) {
	if v == nil {
		return "", "", "", ""
	}
	return nanoWallToRFC3339(v.Provenance.CreatedAt),
		nanoWallToRFC3339(v.Provenance.UpdatedAt),
		nanoWallPtrToRFC3339(v.Validity.ValidFrom),
		nanoWallPtrToRFC3339(v.Validity.ValidTo)
}
