//go:build integration

package channel_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/channel"
	channelstore "github.com/pay-bye/agent-os/internal/storage/postgres/channel"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

func TestExtendUpdatesOnlyActiveLeaseToLaterExpiry(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	insertVocabulary(t, ctx, tx)
	store := channelstore.New(tx)
	now := instant(0)
	enqueue(t, ctx, store, "x32", "x08", now)
	lease := dequeue(t, ctx, store, "x16", now.Add(time.Minute))

	extended, err := store.Extend(ctx, lease.ID(), tokenDigest(), now.Add(2*time.Minute), now.Add(20*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	_, staleErr := store.Extend(ctx, lease.ID(), tokenDigest(), now.Add(3*time.Minute), now.Add(10*time.Minute))
	_, expiredErr := store.Extend(ctx, lease.ID(), tokenDigest(), now.Add(30*time.Minute), now.Add(40*time.Minute))

	if !extended.ExpiresAt().Equal(now.Add(20 * time.Minute)) {
		t.Fatalf("expiry = %s, want %s", extended.ExpiresAt(), now.Add(20*time.Minute))
	}
	if !errors.Is(staleErr, channel.ErrNonIncreasingExtension) {
		t.Fatalf("stale extension error = %v, want non-increasing extension", staleErr)
	}
	if !errors.Is(expiredErr, channel.ErrExpiredLease) {
		t.Fatalf("expired extension error = %v, want expired lease", expiredErr)
	}
}
