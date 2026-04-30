package services

import (
	"context"
	"errors"
	"testing"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

type stubSeamStore struct {
	findCalls int
	lastR     int
	findOut   []domain.SeamHit

	markConflictErr error
	resolveErr      error
}

func (s *stubSeamStore) FindSeams(_ context.Context, q *domain.FindSeamsQuery) ([]domain.SeamHit, error) {
	if s == nil {
		return nil, nil
	}
	s.findCalls++
	s.lastR = q.Radius
	return s.findOut, nil
}

func (s *stubSeamStore) MarkConflict(context.Context, domain.AxialCoord, domain.AxialCoord, string) error {
	if s == nil {
		return nil
	}
	return s.markConflictErr
}

func (s *stubSeamStore) MarkSupersedes(context.Context, domain.AxialCoord, domain.AxialCoord, string) error {
	return nil
}

func (s *stubSeamStore) ResolveSeam(context.Context, string, string, string) error {
	if s == nil {
		return nil
	}
	return s.resolveErr
}

func TestSeamLifecycleService_FindSeams_validation(t *testing.T) {
	t.Parallel()
	st := &stubSeamStore{}
	svc := NewSeamLifecycleService(st)

	t.Run("rejects_negative_radius", func(t *testing.T) {
		t.Parallel()
		_, err := svc.FindSeams(t.Context(), &domain.FindSeamsQuery{
			Center: domain.AxialCoord{Q: 0, R: 0},
			Radius: -1,
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects_large_radius", func(t *testing.T) {
		t.Parallel()
		_, err := svc.FindSeams(t.Context(), &domain.FindSeamsQuery{
			Center: domain.AxialCoord{Q: 0, R: 0},
			Radius: 99,
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("passes_radius", func(t *testing.T) {
		t.Parallel()
		st := &stubSeamStore{}
		svc := NewSeamLifecycleService(st)
		if _, err := svc.FindSeams(t.Context(), &domain.FindSeamsQuery{
			Center: domain.AxialCoord{Q: 1, R: 2},
			Radius: 5,
		}); err != nil {
			t.Fatal(err)
		}
		if st.lastR != 5 {
			t.Fatalf("radius %d", st.lastR)
		}
	})
}

func TestSeamLifecycleService_ResolveSeam_ulid(t *testing.T) {
	t.Parallel()
	svc := NewSeamLifecycleService(&stubSeamStore{resolveErr: errors.New("not found")})
	err := svc.ResolveSeam(t.Context(), "short", "ok", "")
	if err == nil {
		t.Fatal("expected error for bad id")
	}
	valid := "01HZ123456789ABCDEFGHJKMNP" // 26 Crockford chars
	if err := svc.ResolveSeam(t.Context(), valid, "resolved", "note"); err == nil {
		t.Fatal("expected error from store")
	}
}
