package services

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
)

const (
	maxSeamFindRadius   = 64
	maxSeamReasonRunes  = 4096
	maxResolutionNote   = 8192
	ulidLen             = 26
	maxResolutionStatus = 256
)

// SeamLifecycleService implements primary.SeamLifecycle on top of secondary.SeamStore.
type SeamLifecycleService struct {
	store secondary.SeamStore
}

// NewSeamLifecycleService constructs the service with a non-nil store.
func NewSeamLifecycleService(store secondary.SeamStore) *SeamLifecycleService {
	return &SeamLifecycleService{store: store}
}

// FindSeams implements [primary.SeamLifecycle].
func (s *SeamLifecycleService) FindSeams(ctx context.Context, q *domain.FindSeamsQuery) (domain.FindSeamsResponse, error) {
	if s == nil || s.store == nil {
		return domain.FindSeamsResponse{}, fmt.Errorf("seam lifecycle: nil dependencies")
	}
	if q == nil {
		return domain.FindSeamsResponse{}, fmt.Errorf("seam lifecycle: nil query")
	}
	if q.Radius < 0 {
		return domain.FindSeamsResponse{}, fmt.Errorf("seam lifecycle: radius must be >= 0")
	}
	if q.Radius > maxSeamFindRadius {
		return domain.FindSeamsResponse{}, fmt.Errorf("seam lifecycle: radius exceeds max %d", maxSeamFindRadius)
	}
	seams, err := s.store.FindSeams(ctx, q)
	if err != nil {
		return domain.FindSeamsResponse{}, fmt.Errorf("seam lifecycle find seams: %w", err)
	}
	if seams == nil {
		seams = []domain.SeamHit{}
	}
	return domain.FindSeamsResponse{Seams: seams}, nil
}

// MarkConflict implements [primary.SeamLifecycle].
func (s *SeamLifecycleService) MarkConflict(ctx context.Context, cellA, cellB domain.AxialCoord, reason string) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("seam lifecycle: nil dependencies")
	}
	reason, err := normalizeSeamReason(reason)
	if err != nil {
		return fmt.Errorf("seam lifecycle: %w", err)
	}
	if err := s.store.MarkConflict(ctx, cellA, cellB, reason); err != nil {
		return fmt.Errorf("seam lifecycle mark conflict: %w", err)
	}
	return nil
}

// MarkSupersedes implements [primary.SeamLifecycle].
func (s *SeamLifecycleService) MarkSupersedes(ctx context.Context, superseder, superseded domain.AxialCoord, reason string) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("seam lifecycle: nil dependencies")
	}
	reason, err := normalizeSeamReason(reason)
	if err != nil {
		return fmt.Errorf("seam lifecycle: %w", err)
	}
	if err := s.store.MarkSupersedes(ctx, superseder, superseded, reason); err != nil {
		return fmt.Errorf("seam lifecycle mark supersedes: %w", err)
	}
	return nil
}

// ResolveSeam implements [primary.SeamLifecycle].
func (s *SeamLifecycleService) ResolveSeam(ctx context.Context, seamID, resolutionStatus, resolutionNote string) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("seam lifecycle: nil dependencies")
	}
	id := strings.TrimSpace(seamID)
	if len(id) != ulidLen {
		return fmt.Errorf("seam lifecycle: seam_id must be a %d-character ULID", ulidLen)
	}
	status := strings.TrimSpace(resolutionStatus)
	if status == "" {
		return fmt.Errorf("seam lifecycle: resolution_status is required")
	}
	if len(status) > maxResolutionStatus {
		return fmt.Errorf("seam lifecycle: resolution_status too long")
	}
	note := strings.TrimSpace(resolutionNote)
	if len(note) > maxResolutionNote {
		return fmt.Errorf("seam lifecycle: resolution_note too long")
	}
	if err := s.store.ResolveSeam(ctx, id, status, note); err != nil {
		return fmt.Errorf("seam lifecycle resolve seam: %w", err)
	}
	return nil
}

func normalizeSeamReason(reason string) (string, error) {
	r := strings.TrimSpace(reason)
	if utf8.RuneCountInString(r) > maxSeamReasonRunes {
		return "", fmt.Errorf("reason too long (max %d runes)", maxSeamReasonRunes)
	}
	return r, nil
}

// Ensure SeamLifecycleService implements primary.SeamLifecycle.
var _ primary.SeamLifecycle = (*SeamLifecycleService)(nil)
