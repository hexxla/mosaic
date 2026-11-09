package hexxlastore

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type checkedContext struct {
	context.Context
	onCheck func()
}

func (c checkedContext) Err() error {
	err := c.Context.Err()
	c.onCheck()
	return err
}

func TestEmbeddingANNRejectsCancellationWhileQueued(t *testing.T) {
	t.Parallel()
	_, live := openLifecycleDB(t)
	live.mu.Lock()
	release := sync.OnceFunc(live.mu.Unlock)
	defer release()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	checked := make(chan struct{})
	ctx = checkedContext{Context: ctx, onCheck: sync.OnceFunc(func() { close(checked) })}
	done := make(chan error, 1)
	go func() {
		_, err := NewEmbeddingANNAdapter(live).Search(ctx, []float32{1, 0}, 1, 0)
		done <- err
	}()
	<-checked
	cancel()
	release()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("queued search error = %v, want context.Canceled", err)
	}
}
