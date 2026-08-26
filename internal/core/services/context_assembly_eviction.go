package services

import (
	"fmt"

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
	out.TotalTokens = totalBytes
}
