package services

import (
	"context"
	"errors"
	"testing"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
)

type fakeEngine struct {
	out domain.HealthSummary
	err error
}

func (f *fakeEngine) Check(context.Context) (domain.HealthSummary, error) {
	return f.out, f.err
}

func TestHealthService_Status_success_sets_version(t *testing.T) {
	t.Parallel()

	fake := &fakeEngine{out: domain.HealthSummary{CellCount: 3}}
	svc := NewHealthService(fake, "1.2.3")
	got, err := svc.Status(t.Context())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if got.MosaicVersion != "1.2.3" {
		t.Fatalf("MosaicVersion: got %q want 1.2.3", got.MosaicVersion)
	}
	if got.CellCount != 3 {
		t.Fatalf("CellCount: got %d", got.CellCount)
	}
}

func TestHealthService_Status_engine_error(t *testing.T) {
	t.Parallel()

	fake := &fakeEngine{err: errors.New("boom")}
	svc := NewHealthService(fake, "dev")
	_, err := svc.Status(t.Context())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHealthService_Status_nil_engine(t *testing.T) {
	t.Parallel()

	svc := NewHealthService(nil, "dev")
	_, err := svc.Status(t.Context())
	if err == nil {
		t.Fatal("expected error")
	}
}
