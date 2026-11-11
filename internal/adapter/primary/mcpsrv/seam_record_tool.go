package mcpsrv

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
)

// RegisterSeamRecordTool exposes explicit custom seam identity, validity and provenance.
func RegisterSeamRecordTool(server *mcp.Server, svc primary.SeamRecordWriter) error {
	return addTool(server, &mcp.Tool{Name: "mosaic_hexxla_put_seam", Description: "Create or replace a custom seam with a caller-assigned stable ULID, endpoints, validity and provenance. Reuse the same ID on retry to avoid duplicate logical seams; replacement still records a new version. Existing convenience conflict/supersession tools remain available."}, func(ctx context.Context, _ *mcp.CallToolRequest, in domain.PutSeamCommand) (*mcp.CallToolResult, domain.MutationOK, error) {
		out, err := svc.PutSeam(ctx, in)
		return nil, out, err
	})
}
