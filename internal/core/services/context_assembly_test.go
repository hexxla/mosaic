package services_test

import (
	"context"
	"testing"

	"github.com/hexxla/mosaic/internal/core/domain"
	"github.com/hexxla/mosaic/internal/core/services"
)

type stubContextLoader struct {
	last     *domain.LoadContextPackCommand
	response domain.ContextPackResponse
	err      error
}

func (s *stubContextLoader) LoadFromSeeds(_ context.Context, cmd *domain.LoadContextPackCommand) (domain.ContextPackResponse, error) {
	s.last = cmd
	if s.err != nil {
		return domain.ContextPackResponse{}, s.err
	}
	return s.response, nil
}

func TestContextAssemblyService_applies_byte_budget_after_retrieval(t *testing.T) {
	t.Parallel()
	stub := &stubContextLoader{response: domain.ContextPackResponse{
		Cells: []domain.ContextPackCell{
			{Coord: domain.AxialCoord{Q: 0, R: 0}, RawContent: "center", Confidence: 0.9, BudgetBytes: 35, BudgetRing: 0},
			{Coord: domain.AxialCoord{Q: 1, R: 0}, RawContent: "strong", Confidence: 0.8, BudgetBytes: 20, BudgetRing: 1},
			{Coord: domain.AxialCoord{Q: 0, R: 1}, RawContent: "weak", Confidence: 0.2, BudgetBytes: 20, BudgetRing: 1},
		},
		Stats: domain.ContextPackStatsDTO{CandidatesScanned: 3},
	}}
	svc := services.NewContextAssemblyService(stub)
	out, err := svc.LoadFromSeeds(t.Context(), &domain.LoadContextPackCommand{
		Seeds:          seed0(),
		MaxBudgetBytes: 64,
		Explain:        true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Cells) != 2 || out.Cells[0].RawContent != "center" || out.Cells[1].RawContent != "strong" {
		t.Fatalf("cells after eviction: %+v", out.Cells)
	}
	if out.TotalBytes != 55 {
		t.Fatalf("total bytes=%d", out.TotalBytes)
	}
	if out.MaxBudgetBytes != 64 {
		t.Fatalf("budget bytes=%d", out.MaxBudgetBytes)
	}
	if out.Stats.CandidatesScanned != 3 || out.Stats.CellsEvicted != 1 || out.Stats.MaxRingUsed != 1 {
		t.Fatalf("stats: %+v", out.Stats)
	}
	if len(out.Explanations) != 3 {
		t.Fatalf("explanations: %+v", out.Explanations)
	}
}

func TestContextAssemblyService_caps_pack_by_remaining_session_budget(t *testing.T) {
	t.Parallel()
	stub := &stubContextLoader{}
	service := services.NewContextAssemblyService(stub)
	command := domain.LoadContextPackCommand{
		Seeds: seed0(), MaxBudgetBytes: 1000, SessionByteLimit: 64,
	}
	out, err := service.LoadFromSeeds(t.Context(), &command)
	if err != nil {
		t.Fatal(err)
	}
	if out.MaxBudgetBytes != 64 || stub.last.MaxBudgetBytes != 64 {
		t.Fatalf("applied budget = %d, loader budget = %d, want 64", out.MaxBudgetBytes, stub.last.MaxBudgetBytes)
	}
	command.SessionByteLimit = 1
	stub.last = nil
	if _, err := service.LoadFromSeeds(t.Context(), &command); err == nil || stub.last != nil {
		t.Fatalf("expected rejection before loading, got error %v and request %+v", err, stub.last)
	}
}

func TestContextAssemblyService_facet_text_counts_toward_budget(t *testing.T) {
	t.Parallel()
	stub := &stubContextLoader{response: domain.ContextPackResponse{
		Cells: []domain.ContextPackCell{{
			Coord:       domain.AxialCoord{Q: 0, R: 0},
			RawContent:  "small",
			FacetText:   []string{"large derived facet"},
			Confidence:  1,
			BudgetBytes: 70,
		}},
	}}
	svc := services.NewContextAssemblyService(stub)
	out, err := svc.LoadFromSeeds(t.Context(), &domain.LoadContextPackCommand{
		Seeds:          seed0(),
		MaxBudgetBytes: 64,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Cells) != 0 || out.TotalBytes != 0 || out.Stats.CellsEvicted != 1 {
		t.Fatalf("facet budget not enforced: %+v", out)
	}
}

func TestContextAssemblyService_defaults_and_hint(t *testing.T) {
	t.Parallel()
	stub := &stubContextLoader{}
	svc := services.NewContextAssemblyService(stub)
	out, err := svc.LoadFromSeeds(t.Context(), &domain.LoadContextPackCommand{
		Seeds: []domain.AxialCoord{{Q: 0, R: 0}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.RetrievalHint != domain.RetrievalHintAfterContextPack {
		t.Fatalf("hint: %q", out.RetrievalHint)
	}
	const wantRing = 3
	const wantTok = 4096
	if stub.last.MaxRing != wantRing || stub.last.MaxBudgetBytes != wantTok {
		t.Fatalf("defaults: ring=%d tokens=%d", stub.last.MaxRing, stub.last.MaxBudgetBytes)
	}
}

func TestContextAssemblyService_empty_seeds(t *testing.T) {
	t.Parallel()
	svc := services.NewContextAssemblyService(&stubContextLoader{})
	_, err := svc.LoadFromSeeds(t.Context(), &domain.LoadContextPackCommand{Seeds: nil})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestContextAssemblyService_nil(t *testing.T) {
	t.Parallel()
	svc := services.NewContextAssemblyService(nil)
	_, err := svc.LoadFromSeeds(t.Context(), &domain.LoadContextPackCommand{
		Seeds: []domain.AxialCoord{{Q: 1, R: 1}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func seed0() []domain.AxialCoord {
	return []domain.AxialCoord{{Q: 0, R: 0}}
}

func TestContextAssemblyService_budget_resolve(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cmd      domain.LoadContextPackCommand
		wantTok  int
		wantFail bool
	}{
		{name: "default", cmd: domain.LoadContextPackCommand{Seeds: seed0()}, wantTok: 4096},
		{name: "minimum", cmd: domain.LoadContextPackCommand{Seeds: seed0(), MaxBudgetBytes: 64}, wantTok: 64},
		{name: "explicit", cmd: domain.LoadContextPackCommand{Seeds: seed0(), MaxBudgetBytes: 16384}, wantTok: 16384},
		{name: "maximum", cmd: domain.LoadContextPackCommand{Seeds: seed0(), MaxBudgetBytes: 100000}, wantTok: 100000},
		{name: "negative", cmd: domain.LoadContextPackCommand{Seeds: seed0(), MaxBudgetBytes: -1}, wantFail: true},
		{name: "too_small", cmd: domain.LoadContextPackCommand{Seeds: seed0(), MaxBudgetBytes: 63}, wantFail: true},
		{name: "too_large", cmd: domain.LoadContextPackCommand{Seeds: seed0(), MaxBudgetBytes: 100001}, wantFail: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stub := &stubContextLoader{}
			svc := services.NewContextAssemblyService(stub)
			cmd := tt.cmd
			out, err := svc.LoadFromSeeds(t.Context(), &cmd)
			if tt.wantFail {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if stub.last.MaxBudgetBytes != tt.wantTok {
				t.Fatalf("MaxBudgetBytes=%d want %d", stub.last.MaxBudgetBytes, tt.wantTok)
			}
			_ = out
		})
	}
}
