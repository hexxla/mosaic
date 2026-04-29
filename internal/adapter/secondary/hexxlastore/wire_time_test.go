package hexxlastore

import (
	"testing"
	"time"
)

func TestNanoWallToRFC3339(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, time.April, 15, 14, 30, 45, 123456789, time.UTC)
	got := nanoWallToRFC3339(ts.UnixNano())
	if nanoWallToRFC3339(0) != "" {
		t.Fatal("want empty for zero nano")
	}
	if got == "" {
		t.Fatal("want non-empty for valid instant")
	}
}

func TestNanoWallPtrToRFC3339(t *testing.T) {
	t.Parallel()
	v := int64(1)
	if nanoWallPtrToRFC3339(nil) != "" {
		t.Fatal("want empty for nil")
	}
	if nanoWallPtrToRFC3339(&v) == "" {
		t.Fatal("want non-empty for small positive")
	}
}
