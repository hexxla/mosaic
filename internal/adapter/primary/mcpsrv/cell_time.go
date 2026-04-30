package mcpsrv

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

func parseOptionalRFC3339(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, fmt.Errorf("parse RFC3339 time: %w", err)
	}
	return &t, nil
}

func parseAxialCenter(centerQ, centerR *int) (*domain.AxialCoord, error) {
	if centerQ == nil && centerR == nil {
		return nil, nil
	}
	if centerQ == nil || centerR == nil {
		return nil, errors.New("center_q and center_r must both be set or both omitted")
	}
	return &domain.AxialCoord{Q: *centerQ, R: *centerR}, nil
}

func parseCellQuerySort(s string) (domain.CellQuerySort, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return domain.CellQuerySortUnspecified, nil
	}
	switch s {
	case "score", "confidence", "recency", "coord":
		return domain.CellQuerySort(s), nil
	default:
		return domain.CellQuerySortUnspecified, errors.New("sort_by must be one of score, confidence, recency, coord, or empty")
	}
}
