package services

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
)

// SeamRecordService validates custom records while native storage owns seam invariants.
type SeamRecordService struct{ writer secondary.SeamRecordWriter }

// NewSeamRecordService constructs the custom seam workflow.
func NewSeamRecordService(writer secondary.SeamRecordWriter) *SeamRecordService {
	return &SeamRecordService{writer: writer}
}

// PutSeam requires a stable caller-assigned ULID and explicit provenance.
func (s *SeamRecordService) PutSeam(ctx context.Context, q domain.PutSeamCommand) (domain.MutationOK, error) {
	if len(q.ID) != 26 || strings.TrimSpace(q.SourceID) == "" || len(q.SourceID) > 512 || len(q.SeamType) == 0 || len(q.SeamType) > 128 || len(q.Reason) > 65536 || len(q.ResolutionNote) > 65536 {
		return domain.MutationOK{}, fmt.Errorf("custom seam requires a 26-character ULID, source_id, seam_type; text fields exceed limits or required fields are missing")
	}
	if math.IsNaN(q.Confidence) || math.IsInf(q.Confidence, 0) || q.Confidence < 0 || q.Confidence > 1 || math.IsNaN(q.ConfidenceDelta) || math.IsInf(q.ConfidenceDelta, 0) {
		return domain.MutationOK{}, fmt.Errorf("invalid seam confidence")
	}
	for _, instant := range []*time.Time{q.ValidFrom, q.ValidTo, q.DetectedAt, q.CreatedAt, q.UpdatedAt} {
		if err := validateWireTime(instant); err != nil {
			return domain.MutationOK{}, err
		}
	}
	if q.ValidFrom != nil && q.ValidTo != nil && !q.ValidFrom.Before(*q.ValidTo) {
		return domain.MutationOK{}, fmt.Errorf("seam valid_from must precede valid_to")
	}
	return s.writer.PutSeam(ctx, q)
}
