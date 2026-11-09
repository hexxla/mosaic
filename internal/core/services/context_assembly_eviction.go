package services

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// applyContextByteBudget preserves nearest lattice context first, evicting the
// lowest-confidence cell from the outermost occupied ring until the pack fits.
func applyContextByteBudget(out *domain.ContextPackResponse, maxBytes int, explain bool) {
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

	if totalBytes > maxBytes && len(out.Cells) > 0 {
		order := contextEvictionOrder(out.Cells)
		removed := make([]bool, len(out.Cells))
		for _, drop := range order {
			if totalBytes <= maxBytes {
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
			removed[drop] = true
			out.Stats.CellsEvicted++
		}
		kept := out.Cells[:0]
		for i, cell := range out.Cells {
			if !removed[i] {
				kept = append(kept, cell)
			}
		}
		clear(out.Cells[len(kept):])
		out.Cells = kept
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

// contextEvictionOrder computes removal order without repeatedly scanning and
// shifting cells. Original indices break confidence ties and preserve output order.
func contextEvictionOrder(cells []domain.ContextPackCell) []int {
	order := make([]int, len(cells))
	hasNaN := false
	for i := range order {
		order[i] = i
		hasNaN = hasNaN || math.IsNaN(cells[i].Confidence)
	}
	byConfidence := func(a, b int) int {
		return cmp.Or(cmp.Compare(cells[a].Confidence, cells[b].Confidence), cmp.Compare(a, b))
	}
	if !hasNaN {
		slices.SortFunc(order, func(a, b int) int {
			return cmp.Or(cmp.Compare(cells[b].BudgetRing, cells[a].BudgetRing), byConfidence(a, b))
		})
		return order
	}

	// Preserve the original minimum scan even for NaN confidence: NaNs are
	// ignored until one becomes the first remaining cell in its ring.
	slices.SortFunc(order, func(a, b int) int {
		return cmp.Or(cmp.Compare(cells[b].BudgetRing, cells[a].BudgetRing), cmp.Compare(a, b))
	})
	result := make([]int, 0, len(cells))
	removed := make([]bool, len(cells))
	finite := make([]int, 0, len(cells))
	for start := 0; start < len(order); {
		end := start + 1
		for end < len(order) && cells[order[end]].BudgetRing == cells[order[start]].BudgetRing {
			end++
		}
		finite = finite[:0]
		for _, i := range order[start:end] {
			if !math.IsNaN(cells[i].Confidence) {
				finite = append(finite, i)
			}
		}
		slices.SortFunc(finite, byConfidence)
		first, nextFinite := start, 0
		for range end - start {
			for removed[order[first]] {
				first++
			}
			drop := order[first]
			if !math.IsNaN(cells[drop].Confidence) {
				drop = finite[nextFinite]
				nextFinite++
			}
			result = append(result, drop)
			removed[drop] = true
		}
		start = end
	}
	return result
}
