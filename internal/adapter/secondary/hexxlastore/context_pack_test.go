package hexxlastore

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/services"
)

func TestContextPackAdapter_loads_exact_large_radius_and_facet_text(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "context.hexxla")
	db, err := hexxladb.Open(path, &hexxladb.Options{EnableMVCC: true, PageSize: 4096})
	if err != nil {
		t.Fatal(err)
	}
	live := NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })

	center := hexxladb.Coord{Q: 0, R: 0}
	coords := hexxladb.WalkRings(nil, center, 10)
	err = db.Update(func(tx *hexxladb.Tx) error {
		for i, coord := range coords {
			packed, err := hexxladb.Pack(coord)
			if err != nil {
				return err
			}
			rec := hexxladb.NewFactCell(packed, fmt.Sprintf("cell-%d", i), "test", "fact", 0.9)
			if err := tx.PutCell(t.Context(), rec); err != nil {
				return err
			}
		}
		packedCenter, err := hexxladb.Pack(center)
		if err != nil {
			return err
		}
		return tx.PutFacet(hexxladb.NewFacetDerived(packedCenter, 0, "derived", 0))
	})
	if err != nil {
		t.Fatal(err)
	}

	adapter := NewContextPackAdapter(live)
	out, err := adapter.LoadFromSeeds(t.Context(), &domain.LoadContextPackCommand{
		Seeds:            []domain.AxialCoord{{Q: 0, R: 0}},
		MaxRing:          10,
		MaxCells:         1000,
		IncludeFacetText: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Cells) != 331 {
		t.Fatalf("cells=%d want exact radius-10 disk of 331", len(out.Cells))
	}
	if len(out.Cells[0].FacetText) != 1 || out.Cells[0].FacetText[0] != "derived" {
		t.Fatalf("center facet text: %+v", out.Cells[0].FacetText)
	}
	if out.Cells[0].BudgetBytes != len(out.Cells[0].RawContent)+len("derived") {
		t.Fatalf("center budget bytes=%d", out.Cells[0].BudgetBytes)
	}

	service := services.NewContextAssemblyService(adapter)
	budgeted, err := service.LoadFromSeeds(t.Context(), &domain.LoadContextPackCommand{
		Seeds:            []domain.AxialCoord{{Q: 0, R: 0}},
		MaxRing:          10,
		MaxCells:         1000,
		MaxBudgetBytes:   64,
		IncludeFacetText: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if budgeted.TotalBytes > 64 || budgeted.TotalTokens != budgeted.TotalBytes {
		t.Fatalf("budgeted totals: bytes=%d legacy_tokens=%d", budgeted.TotalBytes, budgeted.TotalTokens)
	}
	if len(budgeted.Cells) >= len(out.Cells) || budgeted.Stats.CellsEvicted == 0 {
		t.Fatalf("budget was not applied after retrieval: cells=%d stats=%+v", len(budgeted.Cells), budgeted.Stats)
	}
}

func TestContextPackAdapter_preserves_supersession_seams_and_origin_ring(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "supersession.hexxla")
	db, err := hexxladb.Open(path, &hexxladb.Options{EnableMVCC: true, PageSize: 4096})
	if err != nil {
		t.Fatal(err)
	}
	live := NewLiveDB(db)
	t.Cleanup(func() { _ = live.Close() })

	stale := hexxladb.Coord{Q: 0, R: 0}
	current := hexxladb.Coord{Q: 1, R: 0}
	err = db.Update(func(tx *hexxladb.Tx) error {
		stalePacked, err := hexxladb.Pack(stale)
		if err != nil {
			return err
		}
		currentPacked, err := hexxladb.Pack(current)
		if err != nil {
			return err
		}
		if err := tx.PutCell(t.Context(), hexxladb.NewFactCell(stalePacked, "stale", "test", "fact", 0.5)); err != nil {
			return err
		}
		if err := tx.PutCell(t.Context(), hexxladb.NewFactCell(currentPacked, "current", "test", "fact", 0.9)); err != nil {
			return err
		}
		return tx.MarkSupersedes(current, stale, "updated fact")
	})
	if err != nil {
		t.Fatal(err)
	}

	adapter := NewContextPackAdapter(live)
	out, err := adapter.LoadFromSeeds(t.Context(), &domain.LoadContextPackCommand{
		Seeds:            []domain.AxialCoord{{Q: stale.Q, R: stale.R}},
		MaxRing:          1,
		MaxCells:         10,
		FilterSuperseded: true,
		IncludeSeams:     true,
		Explain:          true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Cells) != 1 || out.Cells[0].RawContent != "current" {
		t.Fatalf("superseded result: %+v", out.Cells)
	}
	if out.Cells[0].BudgetRing != 0 {
		t.Fatalf("replacement ring=%d want original seed ring 0", out.Cells[0].BudgetRing)
	}
	if len(out.Seams) != 1 || out.Seams[0].SeamType != hexxladb.SeamTypeSupersedes {
		t.Fatalf("seams: %+v", out.Seams)
	}
	if len(out.Explanations) != 1 {
		t.Fatalf("explanations: %+v", out.Explanations)
	}
}
