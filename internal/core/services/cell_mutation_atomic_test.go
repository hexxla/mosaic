package services

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

type atomicMutationWriter struct {
	calls      int
	standalone int
	command    *domain.PutCellCommand
	vector     []float32
	result     domain.PutCellMutationResult
	err        error
}

func (*atomicMutationWriter) LookupPutCell(context.Context, *domain.PutCellCommand) (domain.PutCellMutationResult, bool, error) {
	return domain.PutCellMutationResult{}, false, nil
}

func (w *atomicMutationWriter) PutCell(_ context.Context, cmd *domain.PutCellCommand, vector []float32) (domain.PutCellMutationResult, error) {
	w.calls++
	w.command = cmd
	w.vector = slices.Clone(vector)
	return w.result, w.err
}

func (w *atomicMutationWriter) PutEmbedding(context.Context, domain.AxialCoord, []float32, string) error {
	w.standalone++
	return errors.New("atomic cell write must not use standalone PutEmbedding")
}

func (*atomicMutationWriter) DeleteCell(context.Context, *domain.DeleteCellCommand) (bool, error) {
	return false, nil
}

type atomicMutationEmbedder struct {
	calls int
	text  string
	dim   int
	out   []float32
	err   error
}

func (e *atomicMutationEmbedder) Embed(_ context.Context, text string, dim int) ([]float32, error) {
	e.calls++
	e.text, e.dim = text, dim
	return slices.Clone(e.out), e.err
}

func TestPutCellEmbedComputesStoredTextAndUsesOneAtomicWriterCall(t *testing.T) {
	t.Parallel()
	want := domain.PutCellMutationResult{
		OK: true, Embedded: true, Coord: domain.AxialCoord{Q: 3}, Placement: domain.CellPlacementNearAnchor, Probes: 2,
	}
	writer := &atomicMutationWriter{result: want}
	embedder := &atomicMutationEmbedder{out: []float32{0.25, 0.75}}
	service := NewCellMutationService(writer, embedder, 2)
	cmd := &domain.PutCellCommand{
		Coord: domain.AxialCoord{Q: 1}, RawContent: "  exact stored content  ", SourceID: " source ",
		Tags: []string{" topic "}, Placement: domain.CellPlacementNearAnchor, Embed: true, IdempotencyKey: "atomic-test",
	}
	got, err := service.PutCell(t.Context(), cmd)
	if err != nil || got != want {
		t.Fatalf("atomic result = %+v, %v; want %+v", got, err, want)
	}
	if embedder.calls != 1 || embedder.text != cmd.RawContent || embedder.dim != 2 {
		t.Fatalf("embedded text=%q, dimension=%d, calls=%d", embedder.text, embedder.dim, embedder.calls)
	}
	if writer.calls != 1 || writer.standalone != 0 || !slices.Equal(writer.vector, embedder.out) {
		t.Fatalf("writer calls=%d, standalone=%d, vector=%v", writer.calls, writer.standalone, writer.vector)
	}
	if writer.command == cmd || writer.command.SourceID != "source" || writer.command.RawContent != embedder.text || !writer.command.Embed ||
		!slices.Equal(writer.command.Tags, []string{"topic"}) {
		t.Fatalf("normalized atomic command = %+v", writer.command)
	}
	if cmd.SourceID != " source " || cmd.Tags[0] != " topic " || cmd.MaxRadius != 0 {
		t.Fatalf("caller command mutated: %+v", cmd)
	}
}

func TestPutCellWithoutEmbedNeedsNoEmbeddingConfiguration(t *testing.T) {
	t.Parallel()
	writer := &atomicMutationWriter{result: domain.PutCellMutationResult{OK: true}}
	embedder := &atomicMutationEmbedder{err: errors.New("must not call Ollama")}
	service := NewCellMutationService(writer, embedder, 0)
	out, err := service.PutCell(t.Context(), &domain.PutCellCommand{RawContent: "plain", SourceID: "source"})
	if err != nil || !out.OK || out.Embedded || writer.calls != 1 || writer.standalone != 0 || writer.vector != nil || embedder.calls != 0 {
		t.Fatalf("plain put = %+v, %v; cell writes=%d, vector=%v, embed calls=%d", out, err, writer.calls, writer.vector, embedder.calls)
	}
}

func TestPutCellEmbedFailuresNeverReachWriter(t *testing.T) {
	t.Parallel()
	ollamaErr := errors.New("Ollama unavailable")
	for _, test := range []struct {
		name      string
		embedder  *atomicMutationEmbedder
		dimension uint16
		wantCalls int
		wantErr   error
	}{
		{name: "Ollama failure", embedder: &atomicMutationEmbedder{err: ollamaErr}, dimension: 2, wantCalls: 1, wantErr: ollamaErr},
		{name: "no dimension", embedder: &atomicMutationEmbedder{out: []float32{1, 0}}, dimension: 0},
		{name: "no embedder", dimension: 2},
		{name: "wrong dimension", embedder: &atomicMutationEmbedder{out: []float32{1}}, dimension: 2, wantCalls: 1},
		{name: "empty embedding", embedder: &atomicMutationEmbedder{}, dimension: 2, wantCalls: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			writer := &atomicMutationWriter{}
			var service *CellMutationService
			if test.embedder == nil {
				service = NewCellMutationService(writer, nil, test.dimension)
			} else {
				service = NewCellMutationService(writer, test.embedder, test.dimension)
			}
			out, err := service.PutCell(t.Context(), &domain.PutCellCommand{RawContent: "content", SourceID: "source", Embed: true})
			if err == nil || out != (domain.PutCellMutationResult{}) {
				t.Fatalf("failed embedding returned %+v, %v", out, err)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("lost embedder error cause: %v", err)
			}
			if writer.calls != 0 || writer.standalone != 0 {
				t.Fatalf("failed embedding wrote cell=%d, vector=%d", writer.calls, writer.standalone)
			}
			if test.embedder != nil && test.embedder.calls != test.wantCalls {
				t.Fatalf("embedder calls=%d, want %d", test.embedder.calls, test.wantCalls)
			}
		})
	}
}

func TestPutCellWritePolicyRejectsBeforeEmbedding(t *testing.T) {
	t.Parallel()
	denied := errors.New("operator denied capture")
	for _, kind := range []domain.CellPutKind{domain.CellPutKindUserMessage, domain.CellPutKindAssistantResponse} {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			writer := &atomicMutationWriter{}
			embedder := &atomicMutationEmbedder{out: []float32{1, 0}, err: errors.New("must not contact Ollama")}
			service := NewCellMutationService(writer, embedder, 2, WithMutationPolicy(domain.MutationPolicy{
				UserMessageDenied: denied, AssistantResponseDenied: denied,
			}))
			out, err := service.PutCell(t.Context(), &domain.PutCellCommand{Kind: kind, RawContent: "private transcript", SourceID: "session", Embed: true})
			if !errors.Is(err, denied) || out != (domain.PutCellMutationResult{}) || embedder.calls != 0 || writer.calls != 0 || writer.standalone != 0 {
				t.Fatalf("denied put = %+v, %v; embeds=%d, writes=%d, standalone=%d", out, err, embedder.calls, writer.calls, writer.standalone)
			}
		})
	}
}

func TestPutCellAtomicWriterFailurePreservesCauseAndNoSuccess(t *testing.T) {
	t.Parallel()
	writeErr := errors.New("database write rejected")
	writer := &atomicMutationWriter{err: writeErr, result: domain.PutCellMutationResult{OK: true, Embedded: true}}
	embedder := &atomicMutationEmbedder{out: []float32{1, 0}}
	service := NewCellMutationService(writer, embedder, 2)
	out, err := service.PutCell(t.Context(), &domain.PutCellCommand{RawContent: "content", SourceID: "source", Embed: true})
	if !errors.Is(err, writeErr) || out != (domain.PutCellMutationResult{}) || writer.calls != 1 || writer.standalone != 0 || embedder.calls != 1 {
		t.Fatalf("failed atomic put = %+v, %v; writes=%d, standalone=%d, embeds=%d", out, err, writer.calls, writer.standalone, embedder.calls)
	}
}

func TestPutCellKnownDurableErrorPreservesResult(t *testing.T) {
	t.Parallel()
	want := domain.PutCellMutationResult{OK: true, Coord: domain.AxialCoord{Q: 2, R: -1}}
	writer := &atomicMutationWriter{err: domain.ErrMutationCommitDurable, result: want}
	service := NewCellMutationService(writer, nil, 0)
	out, err := service.PutCell(t.Context(), &domain.PutCellCommand{RawContent: "content", SourceID: "source"})
	if !errors.Is(err, domain.ErrMutationCommitDurable) || out != want {
		t.Fatalf("known-durable put = %+v, %v; want %+v with commit error", out, err, want)
	}
}
