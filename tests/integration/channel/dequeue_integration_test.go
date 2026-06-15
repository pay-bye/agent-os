//go:build integration

package channel_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/registry"
	channelstore "github.com/pay-bye/agent-os/internal/storage/postgres/channel"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

func TestDequeueGrantsOneActiveLease(t *testing.T) {
	ctx := context.Background()
	db := postgresfixture.Open(t)
	schema := postgresfixture.CreateSchema(t, ctx, db, "x77")
	setupSchema(t, ctx, db, schema)
	start := make(chan struct{})
	results := make(chan dequeueResult, 2)

	go concurrentDequeue(ctx, t, db, schema, "x16", start, results)
	go concurrentDequeue(ctx, t, db, schema, "x56", start, results)
	close(start)

	successes := 0
	for range 2 {
		result := <-results
		if result.err == nil {
			successes++
			continue
		}
		if !errors.Is(result.err, channel.ErrEmpty) {
			t.Fatalf("dequeue error = %v, want empty queue", result.err)
		}
	}
	if successes != 1 {
		t.Fatalf("successful dequeues = %d, want 1", successes)
	}
}

type dequeueResult struct {
	err error
}

func dequeue(t *testing.T, ctx context.Context, store *channelstore.Store, id string, at time.Time) channel.Lease {
	t.Helper()

	lease, err := store.Dequeue(ctx, registry.ChannelKey("x15"), channel.LeaseRequest{
		ID:          channel.LeaseID(id),
		TokenDigest: tokenDigest(),
		GrantedAt:   at,
		ExpiresAt:   at.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	return lease
}

func concurrentDequeue(
	ctx context.Context,
	t *testing.T,
	db *sql.DB,
	schema string,
	lease string,
	start <-chan struct{},
	results chan<- dequeueResult,
) {
	t.Helper()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		results <- dequeueResult{err: err}
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT set_config('search_path', $1, true)", schema); err != nil {
		results <- dequeueResult{err: err}
		return
	}
	store := channelstore.New(tx)
	<-start
	_, err = store.Dequeue(ctx, registry.ChannelKey("x15"), channel.LeaseRequest{
		ID:          channel.LeaseID(lease),
		TokenDigest: tokenDigest(),
		GrantedAt:   instant(10),
		ExpiresAt:   instant(20),
	})
	if err == nil {
		if err := tx.Commit(); err != nil {
			results <- dequeueResult{err: err}
			return
		}
	}
	results <- dequeueResult{err: err}
}

func tokenDigest() channel.Digest {
	return channel.Token("x-token").Digest()
}
