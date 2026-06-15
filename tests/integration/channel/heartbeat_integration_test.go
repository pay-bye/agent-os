//go:build integration

package channel_test

import (
	"context"
	"testing"
	"time"

	channelstore "github.com/pay-bye/agent-os/internal/storage/postgres/channel"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

func TestHeartbeatReadsActiveLeaseWithoutMutation(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	insertVocabulary(t, ctx, tx)
	store := channelstore.New(tx)
	now := instant(0)
	enqueue(t, ctx, store, "x32", "x08", now)
	lease := dequeue(t, ctx, store, "x16", now.Add(time.Minute))

	heartbeat, err := store.Heartbeat(ctx, lease.ID(), tokenDigest(), now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	unchanged, err := store.Heartbeat(ctx, lease.ID(), tokenDigest(), now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	if heartbeat.ID() != lease.ID() {
		t.Fatalf("heartbeat lease = %q, want %q", heartbeat.ID(), lease.ID())
	}
	if !unchanged.ExpiresAt().Equal(lease.ExpiresAt()) {
		t.Fatalf("expiry changed to %s, want %s", unchanged.ExpiresAt(), lease.ExpiresAt())
	}
}
