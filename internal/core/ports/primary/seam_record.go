package primary

import (
	"context"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// SeamRecordWriter stores a caller-identified custom seam.
type SeamRecordWriter interface {
	PutSeam(context.Context, domain.PutSeamCommand) (domain.MutationOK, error)
}
