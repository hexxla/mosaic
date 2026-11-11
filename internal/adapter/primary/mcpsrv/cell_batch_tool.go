package mcpsrv

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
)

// RegisterCellBatchTool exposes one atomic group of bounded, retry-safe captures.
func RegisterCellBatchTool(server *mcp.Server, svc primary.BatchCellMutation) error {
	type input struct {
		Cells []putCellInput `json:"cells" jsonschema:"1..32 near_anchor writes, each with a distinct idempotency_key; combined text and tags at most 1 MiB"`
	}
	return addTool(server, &mcp.Tool{Name: "mosaic_hexxla_put_cells", Description: "Capture 1..32 cells atomically. All cells require near_anchor placement and distinct idempotency_key values. Validation, policy checks and embedding generation precede one transaction; failure rolls back the entire new group. Retry the same keys and arguments after a lost response to recover original coordinates. Existing receipts can be replayed as part of a group."}, func(ctx context.Context, _ *mcp.CallToolRequest, in input) (*mcp.CallToolResult, domain.PutCellsResult, error) {
		cmd := &domain.PutCellsCommand{Cells: make([]domain.PutCellCommand, len(in.Cells))}
		for i := range in.Cells {
			cmd.Cells[i] = *putCellCommand(&in.Cells[i], domain.CellPutKind(in.Cells[i].Kind))
		}
		out, err := svc.PutCells(ctx, cmd)
		return nil, out, err
	})
}
