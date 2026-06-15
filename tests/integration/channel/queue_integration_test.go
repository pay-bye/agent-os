//go:build integration

package channel_test

import (
	"context"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/registry"
	channelstore "github.com/pay-bye/agent-os/internal/storage/postgres/channel"
	"github.com/pay-bye/agent-os/internal/workitem"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

func TestQueuePreservesFIFOOrder(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	insertVocabulary(t, ctx, tx)
	store := channelstore.New(tx)
	now := instant(0)
	enqueue(t, ctx, store, "x32", "x08", now)
	enqueue(t, ctx, store, "x33", "x09", now.Add(time.Minute))
	enqueue(t, ctx, store, "x59", "x10", now.Add(2*time.Minute))

	first := dequeue(t, ctx, store, "x16", now.Add(3*time.Minute))
	second := dequeue(t, ctx, store, "x56", now.Add(4*time.Minute))
	third := dequeue(t, ctx, store, "x57", now.Add(5*time.Minute))

	requireWorkItems(t, []channel.Lease{first, second, third}, "x08", "x09", "x10")
}

func requireWorkItems(t *testing.T, leases []channel.Lease, want ...string) {
	t.Helper()

	for index, lease := range leases {
		if lease.WorkItem() != workitem.ID(want[index]) {
			t.Fatalf("lease[%d] work item = %q, want %s", index, lease.WorkItem(), want[index])
		}
	}
}

func enqueue(t *testing.T, ctx context.Context, store *channelstore.Store, id string, item string, at time.Time) {
	t.Helper()

	_, err := store.Enqueue(ctx, channel.EntryInput{
		ID:          channel.EntryID(id),
		Channel:     registry.ChannelKey("x15"),
		WorkItem:    workitem.ID(item),
		EnqueuedAt:  at,
		AvailableAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
}
