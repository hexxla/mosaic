package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/sploitzberg/go-llm-project-structure/internal/config"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/ports/primary"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/ports/secondary"
)

const (
	maxTagsPerCell   = 64
	maxSourceIDBytes = 512
)

// CellMutationService implements [primary.CellMutation] with validation and optional Ollama for embed text.
type CellMutationService struct {
	writer    secondary.CellWriter
	embedText secondary.TextEmbedder
	embedDim  uint16
	runtime   config.MosaicRuntimeConfig
}

// CellMutationOption configures [NewCellMutationService].
type CellMutationOption func(*CellMutationService)

// WithMosaicRuntime attaches startup config gates ([config.MosaicRuntimeConfig]) for put_cell policy checks.
func WithMosaicRuntime(c config.MosaicRuntimeConfig) CellMutationOption {
	return func(s *CellMutationService) {
		s.runtime = c
	}
}

// NewCellMutationService constructs CellMutationService. embedText may be nil if only raw vectors are used for PutEmbedding text path.
func NewCellMutationService(w secondary.CellWriter, embedText secondary.TextEmbedder, embedDim uint16, opts ...CellMutationOption) *CellMutationService {
	s := &CellMutationService{writer: w, embedText: embedText, embedDim: embedDim}
	for _, o := range opts {
		o(s)
	}
	return s
}

// PutCell implements [primary.CellMutation].
func (s *CellMutationService) PutCell(ctx context.Context, cmd *domain.PutCellCommand) error {
	if s == nil || s.writer == nil {
		return fmt.Errorf("cell mutation: nil dependencies")
	}
	if cmd == nil {
		return fmt.Errorf("cell mutation: nil command")
	}
	c, err := normalizePutCell(cmd)
	if err != nil {
		return fmt.Errorf("cell mutation: %w", err)
	}
	if err := s.runtime.PutCellDenied(c.Kind); err != nil {
		return fmt.Errorf("cell mutation: %w", err)
	}
	if err := s.writer.PutCell(ctx, c); err != nil {
		return fmt.Errorf("cell mutation put cell: %w", err)
	}
	return nil
}

// PutEmbedding implements [primary.CellMutation].
func (s *CellMutationService) PutEmbedding(ctx context.Context, cmd *domain.PutEmbeddingCommand) error {
	if s == nil || s.writer == nil {
		return fmt.Errorf("cell mutation: nil dependencies")
	}
	if cmd == nil {
		return fmt.Errorf("cell mutation: nil command")
	}
	if s.embedDim == 0 {
		return fmt.Errorf("cell mutation: embeddings disabled on this database (open with EmbeddingDimension)")
	}
	hasText := strings.TrimSpace(cmd.Text) != ""
	hasVec := len(cmd.Vector) > 0
	switch {
	case hasText && hasVec:
		return fmt.Errorf("cell mutation: provide only one of text or vector for embedding")
	case !hasText && !hasVec:
		return fmt.Errorf("cell mutation: need text (Ollama) or vector for embedding")
	}
	var vec []float32
	if hasText {
		if s.embedText == nil {
			return fmt.Errorf("cell mutation: text embedding requires MOSAIC_OLLAMA_URL and MOSAIC_EMBED_MODEL")
		}
		var err error
		vec, err = s.embedText.Embed(ctx, strings.TrimSpace(cmd.Text), int(s.embedDim))
		if err != nil {
			return fmt.Errorf("cell mutation embed: %w", err)
		}
	} else {
		vec = cmd.Vector
		if len(vec) != int(s.embedDim) {
			return fmt.Errorf("cell mutation: vector length %d does not match database embedding dimension %d", len(vec), s.embedDim)
		}
	}
	if err := s.writer.PutEmbedding(ctx, cmd.Coord, vec); err != nil {
		return fmt.Errorf("cell mutation put embedding: %w", err)
	}
	return nil
}

// DeleteCell implements [primary.CellMutation].
func (s *CellMutationService) DeleteCell(ctx context.Context, cmd *domain.DeleteCellCommand) error {
	if s == nil || s.writer == nil {
		return fmt.Errorf("cell mutation: nil dependencies")
	}
	if cmd == nil {
		return fmt.Errorf("cell mutation: nil command")
	}
	if err := s.runtime.DeleteCellDenied(); err != nil {
		return fmt.Errorf("cell mutation: %w", err)
	}
	if err := s.writer.DeleteCell(ctx, cmd); err != nil {
		return fmt.Errorf("cell mutation delete cell: %w", err)
	}
	return nil
}

func normalizePutCell(cmd *domain.PutCellCommand) (*domain.PutCellCommand, error) {
	if strings.TrimSpace(cmd.RawContent) == "" {
		return nil, fmt.Errorf("cell mutation: raw_content is required")
	}
	src := strings.TrimSpace(cmd.SourceID)
	if src == "" {
		return nil, fmt.Errorf("cell mutation: source_id is required")
	}
	if len(src) > maxSourceIDBytes {
		return nil, fmt.Errorf("cell mutation: source_id too long (max %d)", maxSourceIDBytes)
	}
	kind := cmd.Kind
	if kind == "" {
		kind = domain.CellPutKindFact
	}
	switch kind {
	case domain.CellPutKindFact, domain.CellPutKindUserMessage, domain.CellPutKindAssistantResponse:
	default:
		return nil, fmt.Errorf("cell mutation: kind must be %q, %q, or %q",
			domain.CellPutKindFact, domain.CellPutKindUserMessage, domain.CellPutKindAssistantResponse)
	}
	tags := make([]string, 0, len(cmd.Tags))
	for _, t := range cmd.Tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		tags = append(tags, t)
	}
	if len(tags) > maxTagsPerCell {
		return nil, fmt.Errorf("cell mutation: too many tags (max %d)", maxTagsPerCell)
	}
	conf := cmd.Confidence
	if conf < 0 || conf > 1 {
		return nil, fmt.Errorf("cell mutation: confidence must be between 0 and 1")
	}
	out := &domain.PutCellCommand{
		Coord:      cmd.Coord,
		RawContent: cmd.RawContent,
		Tags:       tags,
		SourceID:   src,
		Confidence: conf,
		Kind:       kind,
	}
	return out, nil
}

// Ensure CellMutationService implements primary.CellMutation.
var _ primary.CellMutation = (*CellMutationService)(nil)
