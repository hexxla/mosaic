package mcpsrv

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
)

type browseCursor struct {
	Query   domain.BrowseQuery
	Ring    int
	Offset  int
	Owner   string
	Mode    string
	Expires int64
}

type browseCursorCodec struct{ key []byte }

func newBrowseCursorCodec() *browseCursorCodec { return &browseCursorCodec{key: []byte(rand.Text())} }
func (c *browseCursorCodec) ownerDigest(owner string) string {
	mac := hmac.New(sha256.New, c.key)
	_, _ = mac.Write([]byte("mosaic-cursor-owner-v1\x00" + owner))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func (c *browseCursorCodec) encode(state browseCursor) (string, error) {
	state.Owner = c.ownerDigest(state.Owner)
	data, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, c.key)
	_, _ = mac.Write(data)
	return base64.RawURLEncoding.EncodeToString(data) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
func (c *browseCursorCodec) decode(token, owner, mode string) (browseCursor, error) {
	var state browseCursor
	invalid := errors.New("invalid, expired, or foreign browse cursor; restart enumeration")
	if len(token) > 4096 {
		return state, invalid
	}
	body, signature, ok := strings.Cut(token, ".")
	if !ok {
		return state, invalid
	}
	data, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return state, invalid
	}
	sig, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return state, invalid
	}
	mac := hmac.New(sha256.New, c.key)
	_, _ = mac.Write(data)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return state, invalid
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, invalid
	}
	if owner == "" || state.Owner != c.ownerDigest(owner) || state.Mode != mode || time.Now().Unix() >= state.Expires || state.Query.SnapshotSeq == nil {
		return state, invalid
	}
	state.Owner = owner
	return state, nil
}

type browseInput struct {
	Center         *domain.AxialCoord `json:"center,omitempty"`
	Radius         int                `json:"radius,omitzero" jsonschema:"regional radius 0..128; omitted means exact center"`
	MaxCells       int                `json:"max_cells,omitzero" jsonschema:"page cell/deferred count, default 50, maximum 100"`
	MaxProbes      int                `json:"max_probes,omitzero" jsonschema:"coordinate probes per page, default 1000, maximum 2000"`
	MaxBytes       int                `json:"max_bytes,omitzero" jsonschema:"cell JSON byte budget per page, default 64000, range 1024..100000"`
	IncludeContent bool               `json:"include_content,omitzero"`
	Cursor         string             `json:"cursor,omitempty" jsonschema:"opaque continuation; when supplied omit all other fields"`
}

// RegisterBrowseTools publishes resumable regional export without retaining result sets.
func RegisterBrowseTools(server *mcp.Server, svc primary.MemoryBrowser, budget *RetrievalBudgetTracker) error {
	codec := newBrowseCursorCodec()
	register := func(name, mode, description string) error {
		return addTool(server, &mcp.Tool{Name: name, Description: description}, func(ctx context.Context, req *mcp.CallToolRequest, in browseInput) (*mcp.CallToolResult, domain.BrowsePage, error) {
			out, err := budget.RunBudgetedRead(req, func() (domain.BrowsePage, error) {
				if req == nil || req.Session == nil || req.Session.ID() == "" {
					return domain.BrowsePage{}, errors.New("regional continuation requires an identified MCP session")
				}
				owner := req.Session.ID()
				state := browseCursor{Owner: owner, Mode: mode, Expires: time.Now().Add(5 * time.Minute).Unix()}
				if in.Cursor != "" {
					if in.Center != nil || in.Radius != 0 || in.MaxCells != 0 || in.MaxProbes != 0 || in.MaxBytes != 0 || in.IncludeContent {
						return domain.BrowsePage{}, errors.New("cursor must be supplied without other browse arguments")
					}
					var err error
					state, err = codec.decode(in.Cursor, owner, mode)
					if err != nil {
						return domain.BrowsePage{}, err
					}
				} else {
					if in.Center == nil {
						return domain.BrowsePage{}, errors.New("center is required for initial enumeration")
					}
					state.Query = domain.BrowseQuery{Center: *in.Center, Radius: in.Radius, MaxCells: in.MaxCells, MaxProbes: in.MaxProbes, MaxBytes: in.MaxBytes, IncludeContent: in.IncludeContent || mode == "export"}
				}
				state.Query.Ring = state.Ring
				state.Query.Offset = state.Offset
				if remaining := budget.RemainingBytes(req); remaining > 0 {
					// Reserve room for cursor, arrays and protocol metadata. Return the same
					// request on rejection; no cursor or database state is consumed.
					if remaining < 4096 {
						return domain.BrowsePage{}, errors.New("remaining session budget too small for a browse page")
					}
					limit := remaining - 2048
					if state.Query.MaxBytes == 0 || state.Query.MaxBytes > limit {
						state.Query.MaxBytes = min(64000, limit)
					}
				}
				page, err := svc.Browse(ctx, state.Query)
				if err != nil {
					return domain.BrowsePage{}, err
				}
				if !page.Done {
					state.Ring = page.NextRing
					state.Offset = page.NextOffset
					state.Query.SnapshotSeq = new(page.SnapshotSeq)
					page.NextCursor, err = codec.encode(state)
				}
				return page, err
			})
			return nil, out, err
		})
	}
	return registerTools(
		func() error {
			return register("mosaic_hexxla_browse_cells", "browse", "Enumerate a bounded hex region in stable ring order at one retained snapshot, without collecting the whole result. Cursors are session-bound, expire after five minutes and fail after server restart. Retry the same cursor safely. Empty pages can advance across sparse space. deferred_cells need read_cell_chunk with format=record_json. This is regional browsing, not ranked-query pagination.")
		},
		func() error {
			return register("mosaic_hexxla_export_cells", "export", "Export regional cell pages with content using snapshot-pinned continuation. Collect cells and deferred_cells; use read_cell_chunk(format=record_json, snapshot.seq=snapshot_seq) for complete native records and oversized cells. This transfers cells only, not facets/edges/vectors or a physical backup. No network writer holds a DB transaction.")
		},
		func() error {
			return budget.registerReadTool(server, "mosaic_hexxla_read_cell_chunk", "Read up to 8192 bytes as base64 at a selected snapshot. format=content (default) reads raw bytes; record_json reads a complete native CellRecord JSON document preserving all metadata. Resume at next_offset, verify sha256 after concatenation, and use the same snapshot.", svc.ReadCellChunk)
		},
	)
}
