package services

import (
	"fmt"
	"math"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/hexxla/mosaic/internal/core/domain"
)

func TestContextByteBudgetMatchesRepeatedEviction(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(13, 29)) // #nosec G404 -- Fixed seeds generate reproducible equivalence cases, not secrets.
	for trial := range 2000 {
		pack := domain.ContextPackResponse{
			Stats:        domain.ContextPackStatsDTO{CandidatesScanned: 111, CellsEvicted: 3, MaxRingUsed: 9},
			Explanations: []string{"existing explanation"},
		}
		if trial%3 != 0 {
			pack.Cells = make([]domain.ContextPackCell, rng.IntN(100))
		}
		for i := range pack.Cells {
			confidence := float64(rng.IntN(5)) / 4
			if trial%7 == 0 {
				confidence = []float64{math.NaN(), math.Inf(-1), math.Inf(1), 0.5}[rng.IntN(4)]
			}
			pack.Cells[i] = domain.ContextPackCell{
				Coord: domain.AxialCoord{Q: i}, RawContent: fmt.Sprintf("cell-%d", i),
				Confidence: confidence, BudgetBytes: rng.IntN(40) - 2, BudgetRing: rng.IntN(8) - 2,
			}
		}
		budget := rng.IntN(1000) - 10
		explain := trial%2 == 0
		got, want := cloneEvictionPack(pack), cloneEvictionPack(pack)
		applyContextByteBudget(&got, budget, explain)
		applyContextByteBudgetRepeated(&want, budget, explain)
		assertEvictionPackEqual(t, trial, got, want)
	}
}

func TestContextByteBudgetNilAndEmpty(t *testing.T) {
	t.Parallel()
	applyContextByteBudget(nil, 64, true)
	for _, cells := range [][]domain.ContextPackCell{nil, {}} {
		got := domain.ContextPackResponse{Cells: cells}
		want := cloneEvictionPack(got)
		applyContextByteBudget(&got, 64, false)
		applyContextByteBudgetRepeated(&want, 64, false)
		assertEvictionPackEqual(t, 0, got, want)
	}
}

func assertEvictionPackEqual(t *testing.T, trial int, got, want domain.ContextPackResponse) {
	t.Helper()
	if len(got.Cells) != len(want.Cells) {
		t.Fatalf("trial %d: cells=%d want %d", trial, len(got.Cells), len(want.Cells))
	}
	for i := range got.Cells {
		if math.Float64bits(got.Cells[i].Confidence) != math.Float64bits(want.Cells[i].Confidence) {
			t.Fatalf("trial %d: cell %d confidence differs", trial, i)
		}
		// reflect.DeepEqual deliberately treats NaN as unequal to itself.
		got.Cells[i].Confidence, want.Cells[i].Confidence = 0, 0
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("trial %d: got %+v; want %+v", trial, got, want)
	}
}

func cloneEvictionPack(pack domain.ContextPackResponse) domain.ContextPackResponse {
	pack.Cells = slices.Clone(pack.Cells)
	pack.Explanations = slices.Clone(pack.Explanations)
	return pack
}

func BenchmarkContextByteBudget(b *testing.B) {
	for _, count := range []int{256, 8192} {
		for _, percentKept := range []int{0, 50, 100} {
			for _, explain := range []bool{false, true} {
				pack := domain.ContextPackResponse{Cells: make([]domain.ContextPackCell, count)}
				for i := range pack.Cells {
					pack.Cells[i] = domain.ContextPackCell{
						Coord: domain.AxialCoord{Q: i}, RawContent: "benchmark cell",
						BudgetBytes: 100, BudgetRing: i % 33, Confidence: float64(i%7) / 6,
					}
				}
				budget := count * percentKept
				for _, implementation := range []struct {
					name  string
					apply func(*domain.ContextPackResponse, int, bool)
				}{
					{"repeated", applyContextByteBudgetRepeated},
					{"sorted", applyContextByteBudget},
				} {
					b.Run(fmt.Sprintf("cells=%d/kept=%d/explain=%t/%s", count, percentKept, explain, implementation.name), func(b *testing.B) {
						b.ReportAllocs()
						for b.Loop() {
							out := cloneEvictionPack(pack)
							implementation.apply(&out, budget, explain)
						}
					})
				}
			}
		}
	}
}

// applyContextByteBudgetRepeated is the pre-optimization implementation. Keep it
// independent as an oracle for removal order, explanations, and compatibility.
func applyContextByteBudgetRepeated(out *domain.ContextPackResponse, maxBytes int, explain bool) {
	if out == nil {
		return
	}
	totalBytes := 0
	for i := range out.Cells {
		if out.Cells[i].BudgetBytes == 0 {
			out.Cells[i].BudgetBytes = len(out.Cells[i].RawContent)
		}
		totalBytes += out.Cells[i].BudgetBytes
	}
	for totalBytes > maxBytes && len(out.Cells) > 0 {
		outermostRing := out.Cells[0].BudgetRing
		for i := 1; i < len(out.Cells); i++ {
			outermostRing = max(outermostRing, out.Cells[i].BudgetRing)
		}
		drop := -1
		for i := range out.Cells {
			if out.Cells[i].BudgetRing != outermostRing {
				continue
			}
			if drop == -1 || out.Cells[i].Confidence < out.Cells[drop].Confidence {
				drop = i
			}
		}
		if drop == -1 {
			break
		}
		dropped := out.Cells[drop]
		totalBytes -= dropped.BudgetBytes
		if explain {
			out.Explanations = append(out.Explanations, fmt.Sprintf(
				"(%d,%d) ring=%d evicted_low_confidence bytes=%d",
				dropped.Coord.Q, dropped.Coord.R, dropped.BudgetRing, dropped.BudgetBytes,
			))
		}
		out.Cells = append(out.Cells[:drop], out.Cells[drop+1:]...)
		out.Stats.CellsEvicted++
	}
	out.Stats.MaxRingUsed = 0
	for i := range out.Cells {
		out.Stats.MaxRingUsed = max(out.Stats.MaxRingUsed, out.Cells[i].BudgetRing)
		if explain {
			out.Explanations = append(out.Explanations, fmt.Sprintf(
				"(%d,%d) ring=%d included bytes=%d",
				out.Cells[i].Coord.Q, out.Cells[i].Coord.R,
				out.Cells[i].BudgetRing, out.Cells[i].BudgetBytes,
			))
		}
	}
	out.TotalBytes = totalBytes
}
