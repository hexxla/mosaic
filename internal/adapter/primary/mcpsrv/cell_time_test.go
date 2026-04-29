package mcpsrv

import (
	"testing"
	"time"
)

func TestParseOptionalRFC3339(t *testing.T) {
	t.Parallel()
	got, err := parseOptionalRFC3339("")
	if err != nil || got != nil {
		t.Fatalf("empty: got %v err %v", got, err)
	}
	ts, err := parseOptionalRFC3339("2024-01-15T12:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if ts == nil || !ts.Equal(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("time: got %v", ts)
	}
	_, err = parseOptionalRFC3339("not-a-date")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseAxialCenter(t *testing.T) {
	t.Parallel()
	q, r := 1, -2
	c, err := parseAxialCenter(&q, &r)
	if err != nil || c.Q != 1 || c.R != -2 {
		t.Fatalf("got %+v err %v", c, err)
	}
	c, err = parseAxialCenter(nil, nil)
	if err != nil || c != nil {
		t.Fatalf("nil: %+v err %v", c, err)
	}
	_, err = parseAxialCenter(&q, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseCellQuerySort(t *testing.T) {
	t.Parallel()
	s, err := parseCellQuerySort("confidence")
	if err != nil || s != "confidence" {
		t.Fatalf("got %q err %v", s, err)
	}
	s, err = parseCellQuerySort("")
	if err != nil || s != "" {
		t.Fatalf("empty: got %q err %v", s, err)
	}
	_, err = parseCellQuerySort("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}
