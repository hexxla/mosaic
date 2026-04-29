package services_test

import (
	"context"
	"testing"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/services"
)

type stubContextLoader struct {
	last *domain.LoadContextPackCommand
	err  error
}

func (s *stubContextLoader) LoadFromSeeds(_ context.Context, cmd *domain.LoadContextPackCommand) (domain.ContextPackResponse, error) {
	s.last = cmd
	if s.err != nil {
		return domain.ContextPackResponse{}, s.err
	}
	return domain.ContextPackResponse{}, nil
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
