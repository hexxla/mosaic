package mcpsrv

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/config"
	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
)

type putCellInput struct {
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
}

// RegisterCellMutationTools registers write tools: mosaic_hexxla_put_cell, mosaic_hexxla_put_embedding, mosaic_hexxla_delete_cell.
func RegisterCellMutationTools(server *mcp.Server, svc primary.CellMutation, gates config.MosaicRuntimeConfig, log *slog.Logger) {
	registerPutCellTool(server, svc, gates, log)
	registerPutEmbeddingTool(server, svc, log)
	registerDeleteCellTool(server, svc, gates, log)
}

func registerPutCellTool(server *mcp.Server, svc primary.CellMutation, gates config.MosaicRuntimeConfig, log *slog.Logger) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "mosaic_hexxla_put_cell",
		Description: "Write one cell atomically via HexxlaDB. placement=exact (default) writes at (q,r) and refuses a live coordinate unless allow_overwrite=true. placement=near_anchor treats (q,r) as a caller-chosen semantic anchor and selects the first free coordinate in deterministic ring order within max_radius (default 8, maximum 32). The response always returns the actual coordinate, placement, occupied probes, and whether an exact write replaced a live cell. Mosaic does not infer semantic anchors. Before choosing tags, call mosaic_hexxla_list_tags (and mosaic_hexxla_tag_counts) when taxonomy is unknown — reuse existing tags when relevant instead of inventing near-duplicates. " +
			"kind=fact uses a fact template; user_message / assistant_response match conversational_memory-style tags. source_id is required (provenance source for fact, session id for user/assistant templates).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in putCellInput) (*mcp.CallToolResult, domain.PutCellMutationResult, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_put_cell invoked", "q", in.Q, "r", in.R, "placement", in.Placement)
		}
		kind := domain.CellPutKind(strings.TrimSpace(in.Kind))
		if kind == "" {
			kind = domain.CellPutKindFact
		}
		if err := gates.PutCellDenied(kind); err != nil {
			return nil, domain.PutCellMutationResult{}, fmt.Errorf("mosaic_hexxla_put_cell: %w", err)
		}
		result, err := svc.PutCell(ctx, putCellCommand(&in, kind))
		if err != nil {
			return nil, domain.PutCellMutationResult{}, err
		}
		return nil, result, nil
	})
}

func putCellCommand(in *putCellInput, kind domain.CellPutKind) *domain.PutCellCommand {
	return &domain.PutCellCommand{
		Coord:          domain.AxialCoord{Q: in.Q, R: in.R},
		RawContent:     in.RawContent,
		Tags:           append([]string(nil), in.Tags...),
		SourceID:       in.SourceID,
		Confidence:     in.Confidence,
		Kind:           kind,
		Placement:      domain.CellPlacementMode(strings.TrimSpace(in.Placement)),
		MaxRadius:      in.MaxRadius,
		AllowOverwrite: in.AllowOverwrite,
	}
}

func registerPutEmbeddingTool(server *mcp.Server, svc primary.CellMutation, log *slog.Logger) {
	type putEmbInput struct {
		Q      int       `json:"q" jsonschema:"axial q"`
		R      int       `json:"r" jsonschema:"axial r"`
		Text   string    `json:"text,omitempty" jsonschema:"natural language to embed via Ollama (mutually exclusive with vector)"`
		Vector []float32 `json:"vector,omitempty" jsonschema:"raw float32 embedding; length must match DB EmbeddingDimension"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mosaic_hexxla_put_embedding",
		Description: "Store an embedding vector at axial (q,r) via HexxlaDB Tx.PutEmbedding. Provide either text (embedded with MOSAIC_EMBED_MODEL) or vector (full float32[], length must match the database embedding dimension). Requires embeddings enabled on the DB (same dimension as MOSAIC_DB_PATH database).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in putEmbInput) (*mcp.CallToolResult, domain.MutationOK, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_put_embedding invoked", "q", in.Q, "r", in.R)
		}
		err := svc.PutEmbedding(ctx, &domain.PutEmbeddingCommand{
			Coord:  domain.AxialCoord{Q: in.Q, R: in.R},
			Text:   in.Text,
			Vector: append([]float32(nil), in.Vector...),
		})
		if err != nil {
			return nil, domain.MutationOK{}, err
		}
		return nil, domain.MutationOK{OK: true}, nil
	})
}

func registerDeleteCellTool(server *mcp.Server, svc primary.CellMutation, gates config.MosaicRuntimeConfig, log *slog.Logger) {
	type deleteCellInput struct {
		Q int `json:"q" jsonschema:"axial q"`
		R int `json:"r" jsonschema:"axial r"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "mosaic_hexxla_delete_cell",
		Description: "Delete the cell at axial (q,r) via HexxlaDB Tx.DeleteCell (MVCC tombstone semantics on v2 databases). " +
			"Successful response includes cell_removed: true only when a visible cell existed and was removed; " +
			"false means the coordinate had no live cell (already deleted, wrong coords, or never written) — not an error.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in deleteCellInput) (*mcp.CallToolResult, domain.DeleteCellMutationResult, error) {
		if log != nil {
			log.DebugContext(ctx, "mosaic_hexxla_delete_cell invoked", "q", in.Q, "r", in.R)
		}
		if err := gates.DeleteCellDenied(); err != nil {
			return nil, domain.DeleteCellMutationResult{}, fmt.Errorf("mosaic_hexxla_delete_cell: %w", err)
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
