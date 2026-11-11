package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hexxla/mosaic/internal/core/domain"
	"github.com/hexxla/mosaic/internal/core/ports/primary"
	"github.com/hexxla/mosaic/internal/core/ports/secondary"
)

const (
	maxTagsPerCell                = 64
	maxSourceIDBytes              = 512
	defaultCellPlacementMaxRadius = 8
	maximumCellPlacementMaxRadius = 32
	maxIdempotencyKeyBytes        = 128
)

// CellMutationService implements [primary.CellMutation] with validation and optional Ollama for embed text.
type CellMutationService struct {
	writer         secondary.CellWriter
	embedText      secondary.TextEmbedder
	embedDim       uint16
	embeddingModel string
	policy         domain.MutationPolicy
}

// CellMutationOption configures [NewCellMutationService].
type CellMutationOption func(*CellMutationService)

// WithEmbeddingModel requires caller-declared identity for external vectors.
func WithEmbeddingModel(model string) CellMutationOption {
	return func(s *CellMutationService) { s.embeddingModel = model }
}

// WithMutationPolicy sets the application policy for cell writes.
func WithMutationPolicy(policy domain.MutationPolicy) CellMutationOption {
	return func(s *CellMutationService) {
		s.policy = policy
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
func (s *CellMutationService) PutCell(ctx context.Context, cmd *domain.PutCellCommand) (domain.PutCellMutationResult, error) {
	if s == nil || s.writer == nil {
		return domain.PutCellMutationResult{}, fmt.Errorf("cell mutation: nil dependencies")
	}
	if cmd == nil {
		return domain.PutCellMutationResult{}, fmt.Errorf("cell mutation: nil command")
	}
	c, err := normalizePutCell(cmd)
	if err != nil {
		return domain.PutCellMutationResult{}, fmt.Errorf("cell mutation: %w", err)
	}
	if err := s.policy.PutCellDenied(c.Kind); err != nil {
		return domain.PutCellMutationResult{}, fmt.Errorf("cell mutation: %w", err)
	}
	if c.IdempotencyKey != "" {
		result, found, err := s.writer.LookupPutCell(ctx, c)
		if err != nil {
			return domain.PutCellMutationResult{}, fmt.Errorf("cell mutation check put cell receipt: %w", err)
		}
		if found {
			return result, nil
		}
	}
	var embedding []float32
	if c.Embed {
		if s.embedText == nil || s.embedDim == 0 {
			return domain.PutCellMutationResult{}, fmt.Errorf("cell mutation: embed requires an embedder and a configured database dimension")
		}
		embedding, err = s.embedText.Embed(ctx, c.RawContent, int(s.embedDim))
		if err != nil {
			return domain.PutCellMutationResult{}, fmt.Errorf("cell mutation embed cell: %w", err)
		}
		if len(embedding) != int(s.embedDim) {
			return domain.PutCellMutationResult{}, fmt.Errorf("cell mutation: computed embedding has wrong dimension")
		}
	}
	result, err := s.writer.PutCell(ctx, c, embedding)
	if err != nil {
		if !errors.Is(err, domain.ErrMutationCommitDurable) {
			result = domain.PutCellMutationResult{}
		}
		return result, fmt.Errorf("cell mutation put cell: %w", err)
	}
	return result, nil
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
	expectedHash := strings.TrimSpace(cmd.ExpectedContentSHA256)
	if cmd.ExpectedContentSHA256 != "" {
		decoded, err := hex.DecodeString(expectedHash)
		if err != nil || len(decoded) != sha256.Size {
			return fmt.Errorf("cell mutation: expected_content_sha256 must contain 64 hexadecimal characters")
		}
		expectedHash = strings.ToLower(expectedHash)
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
			return fmt.Errorf("cell mutation: text embedding requires Ollama (policy ollama: base_url/embed_model or env MOSAIC_OLLAMA_URL / MOSAIC_EMBED_MODEL)")
		}
		var err error
		vec, err = s.embedText.Embed(ctx, strings.TrimSpace(cmd.Text), int(s.embedDim))
		if err != nil {
			return fmt.Errorf("cell mutation embed: %w", err)
		}
	} else {
		if s.embeddingModel != "" && cmd.EmbeddingModel != s.embeddingModel {
			return fmt.Errorf("cell mutation: raw vector embedding_model must match %q", s.embeddingModel)
		}
		vec = cmd.Vector
		if len(vec) != int(s.embedDim) {
			return fmt.Errorf("cell mutation: vector length %d does not match database embedding dimension %d", len(vec), s.embedDim)
		}
	}
	if err := s.writer.PutEmbedding(ctx, cmd.Coord, vec, expectedHash); err != nil {
		return fmt.Errorf("cell mutation put embedding: %w", err)
	}
	return nil
}

// DeleteCell implements [primary.CellMutation].
func (s *CellMutationService) DeleteCell(ctx context.Context, cmd *domain.DeleteCellCommand) (cellRemoved bool, err error) {
	if s == nil || s.writer == nil {
		return false, fmt.Errorf("cell mutation: nil dependencies")
	}
	if cmd == nil {
		return false, fmt.Errorf("cell mutation: nil command")
	}
	if err := s.policy.DeleteCellDenied(); err != nil {
		return false, fmt.Errorf("cell mutation: %w", err)
	}
	cellRemoved, err = s.writer.DeleteCell(ctx, cmd)
	if err != nil {
		return false, fmt.Errorf("cell mutation delete cell: %w", err)
	}
	return cellRemoved, nil
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
	case domain.CellPutKindFact, domain.CellPutKindUserMessage, domain.CellPutKindAssistantResponse, domain.CellPutKindSystemPrompt:
	default:
		return nil, fmt.Errorf("cell mutation: kind must be %q, %q, or %q",
			domain.CellPutKindFact, domain.CellPutKindUserMessage, domain.CellPutKindAssistantResponse)
	}
	var tags []string
	if cmd.Tags != nil {
		tags = make([]string, 0, len(cmd.Tags))
	}
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
	placement := cmd.Placement
	if placement == "" {
		placement = domain.CellPlacementExact
	}
	maxRadius := cmd.MaxRadius
	switch placement {
	case domain.CellPlacementExact:
		if cmd.IdempotencyKey != "" {
			return nil, fmt.Errorf("cell mutation: idempotency_key applies only to near_anchor placement")
		}
		if maxRadius != 0 {
			return nil, fmt.Errorf("cell mutation: max_radius applies only to near_anchor placement")
		}
	case domain.CellPlacementNearAnchor:
		if cmd.AllowOverwrite {
			return nil, fmt.Errorf("cell mutation: allow_overwrite cannot be used with near_anchor placement")
		}
		if maxRadius < 0 || maxRadius > maximumCellPlacementMaxRadius {
			return nil, fmt.Errorf("cell mutation: max_radius must be between 0 and %d", maximumCellPlacementMaxRadius)
		}
		if maxRadius == 0 {
			maxRadius = defaultCellPlacementMaxRadius
		}
		if key := cmd.IdempotencyKey; key == "" || strings.TrimSpace(key) != key || len(key) > maxIdempotencyKeyBytes {
			return nil, fmt.Errorf("cell mutation: near_anchor requires a nonempty idempotency_key of at most %d bytes with no surrounding whitespace", maxIdempotencyKeyBytes)
		}
	default:
		return nil, fmt.Errorf("cell mutation: placement must be %q or %q",
			domain.CellPlacementExact, domain.CellPlacementNearAnchor)
	}
	if m := cmd.Metadata; m != nil {
		for _, instant := range []*time.Time{m.ValidFrom, m.ValidTo, m.CreatedAt, m.UpdatedAt} {
			if err := validateWireTime(instant); err != nil {
				return nil, err
			}
		}
		if len(m.FactType) > 512 {
			return nil, fmt.Errorf("fact_type exceeds 512 bytes")
		}

		if m.ClearValidity && (m.ValidFrom != nil || m.ValidTo != nil) {
			return nil, fmt.Errorf("cell mutation: clear_validity conflicts with validity bounds")
		}
		if m.ClearClusterHint && m.ClusterHint != nil {
			return nil, fmt.Errorf("cell mutation: clear_cluster_hint conflicts with cluster_hint")
		}
		if m.ValidFrom != nil && m.ValidTo != nil && !m.ValidFrom.Before(*m.ValidTo) {
			return nil, fmt.Errorf("cell mutation: valid_from must precede valid_to")
		}
	}
	out := &domain.PutCellCommand{
		Metadata:       cmd.Metadata,
		Coord:          cmd.Coord,
		RawContent:     cmd.RawContent,
		Tags:           tags,
		SourceID:       src,
		Confidence:     conf,
		Kind:           kind,
		Placement:      placement,
		MaxRadius:      maxRadius,
		AllowOverwrite: cmd.AllowOverwrite,
		Embed:          cmd.Embed,
		IdempotencyKey: cmd.IdempotencyKey,
	}
	return out, nil
}

// Ensure CellMutationService implements primary.CellMutation.
var _ primary.CellMutation = (*CellMutationService)(nil)

func validateWireTime(instant *time.Time) error {
	if instant != nil && !instant.Equal(time.Unix(0, instant.UnixNano())) {
		return fmt.Errorf("timestamp is outside the nanosecond storage range")
	}
	return nil
}
