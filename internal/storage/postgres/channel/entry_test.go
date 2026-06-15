package channel

import (
	"context"
	"database/sql"
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestEnqueueInsertsChannelEntry(t *testing.T) {
	store := &Store{command: func(_ context.Context, query string, args ...any) (sql.Result, error) {
		requireQuery(t, query, `
INSERT INTO channel_entries (id, channel_key, work_item_id, enqueued_at, available_at)
VALUES ($1, $2, $3, $4, $5)`)
		requireArgs(t, args, "x32", "x15", "x08", instant(0), instant(0))
		return nil, nil
	}}

	_, err := store.Enqueue(context.Background(), channel.EntryInput{
		ID:          channel.EntryID("x32"),
		Channel:     registry.ChannelKey("x15"),
		WorkItem:    workitem.ID("x08"),
		EnqueuedAt:  instant(0),
		AvailableAt: instant(0),
	})
	if err != nil {
		t.Fatal(err)
	}
}
