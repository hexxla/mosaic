package mcpsrv

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
)

type putCellInput struct {
	Embed          bool     `json:"embed,omitempty" jsonschema:"compute an embedding and commit it atomically with the cell; requires configured model and database dimension"`
	Q              int      `json:"q" jsonschema:"exact axial q, or semantic-anchor q when placement=near_anchor"`
	R              int      `json:"r" jsonschema:"exact axial r, or semantic-anchor r when placement=near_anchor"`
	RawContent     string   `json:"raw_content" jsonschema:"cell body text"`
	Tags           []string `json:"tags,omitempty" jsonschema:"optional tag strings (merged with template tags)"`
	SourceID       string   `json:"source_id" jsonschema:"provenance source or session id"`
	Confidence     float64  `json:"confidence" jsonschema:"0..1"`
	Kind           string   `json:"kind,omitempty" jsonschema:"fact (default) | user_message | assistant_response"`
	Placement      string   `json:"placement,omitempty" jsonschema:"exact (default) | near_anchor"`
	MaxRadius      int      `json:"max_radius,omitempty" jsonschema:"near_anchor search radius; default 8, maximum 32"`
	AllowOverwrite bool     `json:"allow_overwrite,omitempty" jsonschema:"exact placement only: explicitly replace a live cell at q,r"`
	IdempotencyKey string   `json:"idempotency_key,omitempty" jsonschema:"required for near_anchor: unique key per logical write, reuse on retries (maximum 128 bytes)"`
}

// RegisterCoreCellMutationTools registers normal cell writes and deletes.
func RegisterCoreCellMutationTools(server *mcp.Server, svc primary.CellMutation, log *slog.Logger) error {
	return registerTools(
		func() error { return registerPutCellTool(server, svc, log) },
		func() error { return registerDeleteCellTool(server, svc, log) },
	)
}

func registerPutCellTool(server *mcp.Server, svc primary.CellMutation, log *slog.Logger) error {
	return addTool(server, &mcp.Tool{
		Name: "mosaic_hexxla_put_cell",
		Description: "Write one cell atomically via HexxlaDB. Set embed:true to compute and commit its embedding in the same transaction; model failure writes nothing. Exact placement refuses deleted coordinates and requires allow_overwrite=true to update a live cell. Near-anchor placement selects the first unused coordinate, skipping live and deleted cells, in deterministic ring order within max_radius (default 8, maximum 32). Changing raw content invalidates the previous embedding. The response returns the actual coordinate, placement, occupied probes, and whether an exact write replaced a live cell. Use mosaic_hexxla_list_tags to reuse relevant tags. " +
			"kind=fact uses a fact template; user_message / assistant_response match conversational_memory-style tags. source_id is required (provenance source for fact, session id for user/assistant templates). " +
			"Near-anchor writes require a unique idempotency_key per logical write. Retry with the same key and arguments to recover the original result. A commit_outcome=durable error confirms the cell was saved; commit_outcome=uncertain requires reopening the database before reconciling or retrying.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in putCellInput) (*mcp.CallToolResult, domain.PutCellMutationResult, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_put_cell invoked", "q", in.Q, "r", in.R, "placement", in.Placement)
		}
		kind := domain.CellPutKind(strings.TrimSpace(in.Kind))
		if kind == "" {
			kind = domain.CellPutKindFact
		}
		result, err := svc.PutCell(ctx, putCellCommand(&in, kind))
		if err != nil {
			if errors.Is(err, domain.ErrMutationCommitDurable) && result.OK {
				err = fmt.Errorf("cell saved at (%d,%d): %w", result.Coord.Q, result.Coord.R, err)
			}
			return nil, domain.PutCellMutationResult{}, err
		}
		return nil, result, nil
	})
}

func putCellCommand(in *putCellInput, kind domain.CellPutKind) *domain.PutCellCommand {
	return &domain.PutCellCommand{
		Coord:          domain.AxialCoord{Q: in.Q, R: in.R},
		RawContent:     in.RawContent,
		Tags:           slices.Clone(in.Tags),
		SourceID:       in.SourceID,
		Confidence:     in.Confidence,
		Kind:           kind,
		Placement:      domain.CellPlacementMode(strings.TrimSpace(in.Placement)),
		MaxRadius:      in.MaxRadius,
		AllowOverwrite: in.AllowOverwrite,
		Embed:          in.Embed,
		IdempotencyKey: in.IdempotencyKey,
	}
}

// RegisterPutEmbeddingTool registers advanced standalone vector maintenance.
func RegisterPutEmbeddingTool(server *mcp.Server, svc primary.CellMutation, log *slog.Logger) error {
	type putEmbInput struct {
		Q                     int       `json:"q" jsonschema:"axial q"`
		R                     int       `json:"r" jsonschema:"axial r"`
		Text                  string    `json:"text,omitempty" jsonschema:"natural language to embed via Ollama (mutually exclusive with vector)"`
		Vector                []float32 `json:"vector,omitempty" jsonschema:"raw float32 embedding; length must match DB EmbeddingDimension"`
		ExpectedContentSHA256 string    `json:"expected_content_sha256,omitempty" jsonschema:"optional 64-hex SHA256 digest of current raw_content; rejects a changed cell atomically; omit for unconditional embedding replacement"`
	}

	return addTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_put_embedding",
		Description: "Store an embedding vector for a live cell at axial (q,r). Provide either text (embedded with MOSAIC_EMBED_MODEL) or a vector whose length matches the database embedding dimension. Set expected_content_sha256 to the SHA256 of the cell raw_content you embedded to reject concurrent content changes; omission preserves unconditional replacement.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in putEmbInput) (*mcp.CallToolResult, domain.MutationOK, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_put_embedding invoked", "q", in.Q, "r", in.R)
		}
		err := svc.PutEmbedding(ctx, &domain.PutEmbeddingCommand{
			Coord:                 domain.AxialCoord{Q: in.Q, R: in.R},
			Text:                  in.Text,
			Vector:                slices.Clone(in.Vector),
			ExpectedContentSHA256: in.ExpectedContentSHA256,
		})
		if err != nil {
			return nil, domain.MutationOK{}, err
		}
		return nil, domain.MutationOK{OK: true}, nil
	})
}

func registerDeleteCellTool(server *mcp.Server, svc primary.CellMutation, log *slog.Logger) error {
	type deleteCellInput struct {
		Q int `json:"q" jsonschema:"axial q"`
		R int `json:"r" jsonschema:"axial r"`
	}

	return addTool(server, &mcp.Tool{
		Name: "mosaic_hexxla_delete_cell",
		Description: "Delete the cell at axial (q,r) via HexxlaDB Tx.DeleteCell (MVCC tombstone semantics on v2 databases). " +
			"Successful response includes cell_removed: true only when a visible cell existed and was removed; " +
			"false means the coordinate had no live cell (already deleted, wrong coords, or never written) — not an error.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in deleteCellInput) (*mcp.CallToolResult, domain.DeleteCellMutationResult, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_delete_cell invoked", "q", in.Q, "r", in.R)
		}
		removed, err := svc.DeleteCell(ctx, &domain.DeleteCellCommand{
			Coord: domain.AxialCoord{Q: in.Q, R: in.R},
		})
		if err != nil {
			return nil, domain.DeleteCellMutationResult{}, err
		}
		return nil, domain.DeleteCellMutationResult{OK: true, CellRemoved: removed}, nil
	})
}
