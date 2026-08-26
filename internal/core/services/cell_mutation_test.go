package services

import (
	"context"
	"testing"

	"github.com/sploitzberg/mosaic/internal/config"
	"github.com/sploitzberg/mosaic/internal/core/domain"
)

type stubCellWriter struct {
	putCellCalls    int
	putEmbCoords    domain.AxialCoord
	putEmbVecLen    int
	deleteCellCalls int
	lastPutCellKind domain.CellPutKind
	lastPutCell     *domain.PutCellCommand
	putCellResult   domain.PutCellMutationResult
}

func (s *stubCellWriter) PutCell(_ context.Context, cmd *domain.PutCellCommand) (domain.PutCellMutationResult, error) {
	if s == nil {
		return domain.PutCellMutationResult{}, nil
	}
	s.putCellCalls++
	if cmd != nil {
		s.lastPutCellKind = cmd.Kind
		copied := *cmd
		s.lastPutCell = &copied
	}
	return s.putCellResult, nil
}

func (s *stubCellWriter) PutEmbedding(_ context.Context, coord domain.AxialCoord, vec []float32) error {
	if s == nil {
		return nil
	}
	s.putEmbCoords = coord
	s.putEmbVecLen = len(vec)
	return nil
}

func (s *stubCellWriter) DeleteCell(context.Context, *domain.DeleteCellCommand) (bool, error) {
	if s == nil {
		return false, nil
	}
	s.deleteCellCalls++
	return true, nil
}

type stubEmbedder struct {
	out   []float32
	err   error
	calls int
}

func (e *stubEmbedder) Embed(_ context.Context, _ string, expectedDim int) ([]float32, error) {
	if e == nil {
		return nil, nil
	}
	e.calls++
	if e.err != nil {
		return nil, e.err
	}
	if len(e.out) == expectedDim {
		return append([]float32(nil), e.out...), nil
	}
	v := make([]float32, expectedDim)
	copy(v, e.out)
	return v, nil
}

func TestCellMutationService_PutCell_validation(t *testing.T) {
	t.Parallel()
	t.Run("rejects_empty_content", func(t *testing.T) {
		t.Parallel()
		svc := NewCellMutationService(&stubCellWriter{}, nil, 384)
		_, err := svc.PutCell(t.Context(), &domain.PutCellCommand{
			Coord:      domain.AxialCoord{Q: 0, R: 0},
			RawContent: "   ",
			SourceID:   "src",
			Confidence: 1,
			Kind:       domain.CellPutKindFact,
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("defaults_kind_fact", func(t *testing.T) {
		t.Parallel()
		w := &stubCellWriter{}
		svc := NewCellMutationService(w, nil, 384)
		if _, err := svc.PutCell(t.Context(), &domain.PutCellCommand{
			Coord:      domain.AxialCoord{Q: 1, R: -1},
			RawContent: "hello",
			SourceID:   "session-1",
			Confidence: 0.9,
			Kind:       "",
		}); err != nil {
			t.Fatal(err)
		}
		if w.lastPutCellKind != domain.CellPutKindFact {
			t.Fatalf("kind: got %q", w.lastPutCellKind)
		}
		if w.lastPutCell.Placement != domain.CellPlacementExact {
			t.Fatalf("placement: got %q", w.lastPutCell.Placement)
		}
	})

	t.Run("persistence_policy_reject_assistant_when_user_only", func(t *testing.T) {
		t.Parallel()
		pol := config.RetentionPolicy{
			Version:     1,
			CaptureMode: config.CaptureModeUserOnly,
			Enforcement: config.PolicyEnforcementReject,
		}
		w := &stubCellWriter{}
		svc := NewCellMutationService(w, nil, 384, WithMosaicRuntime(config.NewMosaicRuntimeConfig(pol, false)))
		_, err := svc.PutCell(t.Context(), &domain.PutCellCommand{
			Coord:      domain.AxialCoord{Q: 0, R: 0},
			RawContent: "reply",
			SourceID:   "s",
			Confidence: 1,
			Kind:       domain.CellPutKindAssistantResponse,
		})
		if err == nil {
			t.Fatal("expected policy error")
		}
		if w.putCellCalls != 0 {
			t.Fatalf("writer called %d times", w.putCellCalls)
		}
	})

	t.Run("delete_cell_denied_when_config_disallows", func(t *testing.T) {
		t.Parallel()
		w := &stubCellWriter{}
		rt := config.NewMosaicRuntimeConfig(config.DefaultRetentionPolicy(), false)
		svc := NewCellMutationService(w, nil, 384, WithMosaicRuntime(rt))
		_, err := svc.DeleteCell(t.Context(), &domain.DeleteCellCommand{
			Coord: domain.AxialCoord{Q: 0, R: 0},
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if w.deleteCellCalls != 0 {
			t.Fatalf("delete should not reach writer")
		}
	})
}

func TestCellMutationService_PutCell_placement(t *testing.T) {
	t.Parallel()

	t.Run("near_anchor_defaults_radius_and_returns_actual_placement", func(t *testing.T) {
		t.Parallel()
		want := domain.PutCellMutationResult{
			OK:        true,
			Coord:     domain.AxialCoord{Q: 4, R: -2},
			Placement: domain.CellPlacementNearAnchor,
			Probes:    3,
		}
		writer := &stubCellWriter{putCellResult: want}
		svc := NewCellMutationService(writer, nil, 384)
		got, err := svc.PutCell(t.Context(), &domain.PutCellCommand{
			Coord:      domain.AxialCoord{Q: 3, R: -2},
			RawContent: "hello",
			SourceID:   "source",
			Confidence: 0.8,
			Placement:  domain.CellPlacementNearAnchor,
		})
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("result: got %+v want %+v", got, want)
		}
		if writer.lastPutCell.MaxRadius != defaultCellPlacementMaxRadius {
			t.Fatalf("MaxRadius=%d want %d", writer.lastPutCell.MaxRadius, defaultCellPlacementMaxRadius)
		}
	})

	for _, tc := range []struct {
		name string
		cmd  domain.PutCellCommand
	}{
		{
			name: "near_anchor_rejects_overwrite",
			cmd: domain.PutCellCommand{
				Placement: domain.CellPlacementNearAnchor, AllowOverwrite: true,
			},
		},
		{
			name: "near_anchor_rejects_excessive_radius",
			cmd: domain.PutCellCommand{
				Placement: domain.CellPlacementNearAnchor, MaxRadius: maximumCellPlacementMaxRadius + 1,
			},
		},
		{
			name: "exact_rejects_radius",
			cmd: domain.PutCellCommand{
				Placement: domain.CellPlacementExact, MaxRadius: 1,
			},
		},
		{
			name: "rejects_unknown_placement",
			cmd: domain.PutCellCommand{
				Placement: "automatic",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			writer := &stubCellWriter{}
			svc := NewCellMutationService(writer, nil, 384)
			cmd := tc.cmd
			cmd.RawContent = "hello"
			cmd.SourceID = "source"
			cmd.Confidence = 0.8
			if _, err := svc.PutCell(t.Context(), &cmd); err == nil {
				t.Fatal("expected placement validation error")
			}
			if writer.putCellCalls != 0 {
				t.Fatalf("writer called %d times", writer.putCellCalls)
			}
		})
	}
}

func TestCellMutationService_PutEmbedding_text_and_vector_paths(t *testing.T) {
	t.Parallel()
	t.Run("embed_text", func(t *testing.T) {
		t.Parallel()
		w := &stubCellWriter{}
		emb := &stubEmbedder{out: []float32{1, 0, 0, 1}}
		svc := NewCellMutationService(w, emb, 4)

		ctx := t.Context()
		err := svc.PutEmbedding(ctx, &domain.PutEmbeddingCommand{
			Coord: domain.AxialCoord{Q: 0, R: 0},
			Text:  "probe",
		})
		if err != nil {
			t.Fatal(err)
		}
		if emb.calls != 1 {
			t.Fatalf("embed calls: %d", emb.calls)
		}
		if w.putEmbVecLen != 4 {
			t.Fatalf("vec len %d", w.putEmbVecLen)
		}
	})

	t.Run("raw_vector_no_embedder", func(t *testing.T) {
		t.Parallel()
		w := &stubCellWriter{}
		svc := NewCellMutationService(w, nil, 2)
		v := []float32{0.5, -0.5}
		err := svc.PutEmbedding(t.Context(), &domain.PutEmbeddingCommand{
			Coord:  domain.AxialCoord{Q: 2, R: -1},
			Vector: v,
		})
		if err != nil {
			t.Fatal(err)
		}
		if w.putEmbCoords != (domain.AxialCoord{Q: 2, R: -1}) {
			t.Fatalf("coord %+v", w.putEmbCoords)
		}
	})
}

func TestCellMutationService_PutEmbedding_errors(t *testing.T) {
	t.Parallel()
	w := &stubCellWriter{}
	svc := NewCellMutationService(w, nil, 0)

	err := svc.PutEmbedding(t.Context(), &domain.PutEmbeddingCommand{
		Coord:  domain.AxialCoord{Q: 0, R: 0},
		Vector: []float32{1},
	})
	if err == nil {
		t.Fatal("expected error when embeddings disabled (dim 0)")
	}

	svc2 := NewCellMutationService(w, nil, 384)
	err = svc2.PutEmbedding(t.Context(), &domain.PutEmbeddingCommand{
		Text: "x",
	})
	if err == nil {
		t.Fatal("expected error when embedder nil and text supplied")
	}
}
