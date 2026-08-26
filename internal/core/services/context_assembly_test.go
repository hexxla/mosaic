package services_test

import (
	"context"
	"testing"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/services"
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
	if out.TotalBytes != 55 || out.TotalTokens != 55 {
		t.Fatalf("totals: bytes=%d legacy_tokens=%d", out.TotalBytes, out.TotalTokens)
	}
	if out.MaxBudgetBytes != 64 || out.MaxTokensBudget != 64 {
		t.Fatalf("budgets: bytes=%d legacy_tokens=%d", out.MaxBudgetBytes, out.MaxTokensBudget)
	}
	if out.Stats.CandidatesScanned != 3 || out.Stats.CellsEvicted != 1 || out.Stats.MaxRingUsed != 1 {
		t.Fatalf("stats: %+v", out.Stats)
	}
	if len(out.Explanations) != 3 {
		t.Fatalf("explanations: %+v", out.Explanations)
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
	if stub.last.MaxRing != wantRing || stub.last.MaxTokens != wantTok {
		t.Fatalf("defaults: ring=%d tokens=%d", stub.last.MaxRing, stub.last.MaxTokens)
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
		{
			name: "omit_budget",
			cmd: domain.LoadContextPackCommand{
				Seeds: seed0(), OmitBudget: true,
			},
			wantTok: 100000,
		},
		{
			name: "explicit_legacy_max_tokens",
			cmd: domain.LoadContextPackCommand{
				Seeds: seed0(), MaxTokens: 8192,
			},
			wantTok: 8192,
		},
		{
			name: "explicit_max_budget_bytes",
			cmd: domain.LoadContextPackCommand{
				Seeds: seed0(), MaxBudgetBytes: 16384,
			},
			wantTok: 16384,
		},
		{
			name: "approx_tokens",
			cmd: domain.LoadContextPackCommand{
				Seeds: seed0(), BudgetTokensApprox: 1000, BytesPerApproxToken: 4,
			},
			wantTok: 4000,
		},
		{
			name: "omit_and_token_conflict",
			cmd: domain.LoadContextPackCommand{
				Seeds: seed0(), OmitBudget: true, BudgetTokensApprox: 10,
			},
			wantFail: true,
		},
		{
			name: "token_and_explicit_conflict",
			cmd: domain.LoadContextPackCommand{
				Seeds: seed0(), BudgetTokensApprox: 100, MaxTokens: 5000,
			},
			wantFail: true,
		},
		{
			name: "explicit_too_small",
			cmd: domain.LoadContextPackCommand{
				Seeds: seed0(), MaxTokens: 10,
			},
			wantFail: true,
		},
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
			if stub.last.MaxTokens != tt.wantTok {
				t.Fatalf("MaxTokens=%d want %d", stub.last.MaxTokens, tt.wantTok)
			}
			_ = out
		})
	}
}
