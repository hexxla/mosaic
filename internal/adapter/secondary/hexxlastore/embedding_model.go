package hexxlastore

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
)

const embeddingModelKey = "app/mosaic/embedding-model"

// ErrEmbeddingModelMismatch means the database's vectors cannot be used with
// the configured model until an explicit re-embedding.
var ErrEmbeddingModelMismatch = errors.New("embedding model is incompatible with the database")

// EnsureEmbeddingModel checks the persisted model and binds an empty vector store
// to model. Databases with embeddings of unknown origin require re-embedding.
func EnsureEmbeddingModel(db *hexxladb.DB, model string) error {
	if db.EmbeddingDimension() == 0 {
		return nil
	}
	if model == "" {
		return errors.New("embedding model must be set")
	}
	var found bool
	if err := db.View(func(tx *hexxladb.Tx) error {
		var err error
		found, err = checkEmbeddingModel(tx, model)
		return err
	}); err != nil || found {
		return err
	}
	return db.Update(func(tx *hexxladb.Tx) error {
		found, err := checkEmbeddingModel(tx, model)
		if err != nil || found {
			return err
		}
		count, err := countEmbeddingRows(tx, 1)
		if err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("%w: existing vectors have no recorded model; run mosaic-mcp -reembed with the intended model", ErrEmbeddingModelMismatch)
		}
		return tx.Put([]byte(embeddingModelKey), []byte(model))
	})
}

func checkEmbeddingModel(tx *hexxladb.Tx, model string) (bool, error) {
	stored, found, err := tx.Get([]byte(embeddingModelKey))
	if err != nil || !found {
		return false, err
	}
	if string(stored) != model {
		return true, fmt.Errorf("%w: stored %q, configured %q; use the stored model or run mosaic-mcp -reembed", ErrEmbeddingModelMismatch, stored, model)
	}
	return true, nil
}

// Reembed replaces the live-cell vectors and their model identity atomically.
// Run it offline with exclusive ownership of the open database.
func Reembed(ctx context.Context, db *hexxladb.DB, model string, embedder secondary.TextEmbedder) error {
	dim := int(db.EmbeddingDimension())
	if dim == 0 {
		return errors.New("re-embedding requires a database with a configured embedding dimension")
	}
	if model == "" || embedder == nil {
		return errors.New("re-embedding requires a model and an embedder")
	}
	return db.Update(func(tx *hexxladb.Tx) error {
		var replaced int
		err := tx.ReindexEmbeddings(ctx, func(ctx context.Context, cell hexxladb.CellRecord) ([]float32, error) {
			vec, err := embedder.Embed(ctx, cell.RawContent, dim)
			if err != nil {
				return nil, err
			}
			if len(vec) != dim {
				return nil, fmt.Errorf("%w: model %q returned %d components, want %d", hexxladb.ErrEmbeddingDimension, model, len(vec), dim)
			}
			replaced++
			return vec, nil
		})
		if err != nil {
			return err
		}
		count, err := countEmbeddingRows(tx, replaced+1)
		if err != nil {
			return err
		}
		if count != replaced {
			return fmt.Errorf("re-embedding found vector rows without live cells; repair the database before changing the model")
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		return tx.Put([]byte(embeddingModelKey), []byte(model))
	})
}

// The pinned HexxlaDB API has no vector count. Scan only the documented
// embed/ family and use the typed reindex API for vector changes.
func countEmbeddingRows(tx *hexxladb.Tx, limit int) (int, error) {
	prefix := []byte("embed/")
	var count int
	err := tx.AscendRange(prefix, []byte("embed0"), func(key, _ []byte) bool {
		if !bytes.HasPrefix(key, prefix) {
			return false
		}
		count++
		return count < limit
	})
	return count, err
}
