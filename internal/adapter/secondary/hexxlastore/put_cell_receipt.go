package hexxlastore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"slices"

	"github.com/hexxla/hexxladb"

	"github.com/hexxla/mosaic/internal/core/domain"
)

var errPutCellReplay = errors.New("put_cell receipt already exists")

// The v1 fingerprint names the request fields explicitly so future command
// fields cannot invalidate existing receipts.
type putCellFingerprint struct {
	Metadata   *domain.CellMetadata `json:",omitzero"`
	Q, R       int
	RawContent string
	Tags       []string
	SourceID   string
	Confidence float64
	Kind       domain.CellPutKind
	MaxRadius  int
	Embed      bool
}

type putCellReceipt struct {
	Fingerprint [32]byte
	Result      domain.PutCellMutationResult
}

type putCellRequest struct {
	key         []byte
	fingerprint [32]byte
}

func newPutCellRequest(cmd *domain.PutCellCommand) (putCellRequest, error) {
	tags := slices.Clone(cmd.Tags)
	slices.Sort(tags)
	payload, err := json.Marshal(putCellFingerprint{
		Metadata: cmd.Metadata, Q: cmd.Coord.Q, R: cmd.Coord.R, RawContent: cmd.RawContent, Tags: slices.Compact(tags),
		SourceID: cmd.SourceID, Confidence: cmd.Confidence, Kind: cmd.Kind, MaxRadius: cmd.MaxRadius, Embed: cmd.Embed,
	})
	if err != nil {
		return putCellRequest{}, fmt.Errorf("encode put_cell fingerprint: %w", err)
	}
	keyHash := sha256.Sum256([]byte(cmd.IdempotencyKey))
	return putCellRequest{
		key:         hex.AppendEncode([]byte("app/mosaic/put-cell/v1/"), keyHash[:]),
		fingerprint: sha256.Sum256(payload),
	}, nil
}

func (r putCellRequest) lookup(tx *hexxladb.Tx) (domain.PutCellMutationResult, bool, error) {
	value, found, err := tx.Get(r.key)
	if err != nil || !found {
		return domain.PutCellMutationResult{}, false, err
	}
	var receipt putCellReceipt
	if err := json.Unmarshal(value, &receipt); err != nil {
		return domain.PutCellMutationResult{}, false, fmt.Errorf("decode put_cell receipt: %w", err)
	}
	if receipt.Fingerprint != r.fingerprint {
		return domain.PutCellMutationResult{}, false, domain.ErrIdempotencyKeyConflict
	}
	return receipt.Result, true, nil
}

func (r putCellRequest) store(tx *hexxladb.Tx, result domain.PutCellMutationResult) error {
	value, err := json.Marshal(putCellReceipt{Fingerprint: r.fingerprint, Result: result})
	if err != nil {
		return fmt.Errorf("encode put_cell receipt: %w", err)
	}
	return tx.Put(r.key, value)
}

// LookupPutCell checks a durable receipt before the service computes a vector.
func (a *CellWriterAdapter) LookupPutCell(ctx context.Context, cmd *domain.PutCellCommand) (domain.PutCellMutationResult, bool, error) {
	if a == nil || a.live == nil || cmd == nil {
		return domain.PutCellMutationResult{}, false, errors.New("hexxlastore put cell receipt: missing database or command")
	}
	request, err := newPutCellRequest(cmd)
	if err != nil {
		return domain.PutCellMutationResult{}, false, err
	}
	var result domain.PutCellMutationResult
	var found bool
	err = a.live.WithRead(func(db *hexxladb.DB) error {
		return db.View(func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			var err error
			result, found, err = request.lookup(tx)
			return err
		})
	})
	return result, found, classifyMutationCommit(err)
}
