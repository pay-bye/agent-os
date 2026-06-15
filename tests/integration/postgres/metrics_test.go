//go:build integration

package postgres_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	metricstore "github.com/pay-bye/agent-os/internal/storage/postgres/metrics"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

func TestMetricsAggregatesQueueAndLeasePressure(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	postgresfixture.SetSearchPath(t, ctx, tx, "pg_temp")
	insertChannel(t, ctx, tx)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	insertEntry(t, ctx, tx, entryInput{id: "x01", availableAt: now.Add(-time.Minute)})
	insertEntry(t, ctx, tx, entryInput{id: "x02", availableAt: now.Add(time.Minute)})
	insertEntry(t, ctx, tx, entryInput{id: "x03", availableAt: now.Add(-time.Minute)})
	insertEntry(t, ctx, tx, entryInput{id: "x04", availableAt: now.Add(-time.Minute)})
	insertLease(t, ctx, tx, leaseInput{id: "x13", entry: "x03", expiresAt: now.Add(time.Minute)})
	insertLease(t, ctx, tx, leaseInput{id: "x14", entry: "x04", expiresAt: now.Add(-time.Minute)})

	got, err := metricstore.New(tx).Aggregates(ctx, now)
	if err != nil {
		t.Fatal(err)
	}

	if got.AvailableDepth != 2 {
		t.Fatalf("available depth = %d, want 2", got.AvailableDepth)
	}
	if got.LeasesHeld != 1 {
		t.Fatalf("leases held = %d, want 1", got.LeasesHeld)
	}
	if got.LeasesExpired != 1 {
		t.Fatalf("leases expired = %d, want 1", got.LeasesExpired)
	}
}

type entryInput struct {
	id          string
	availableAt time.Time
}

type leaseInput struct {
	id        string
	entry     string
	expiresAt time.Time
}

type executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func insertChannel(t *testing.T, ctx context.Context, tx executor) {
	t.Helper()

	_, err := tx.ExecContext(ctx, `
INSERT INTO schema_documents (key, document) VALUES ('x91', '{"title":"x91"}');
INSERT INTO need_kinds (key, schema_key, description) VALUES ('x12', 'x91', 'x91');
INSERT INTO nodes (key, description) VALUES ('x17', 'x91');
INSERT INTO channels (key, node_key, description) VALUES ('x15', 'x17', 'x91');
`)
	if err != nil {
		t.Fatal(err)
	}
}

func insertEntry(t *testing.T, ctx context.Context, tx executor, input entryInput) {
	t.Helper()

	_, err := tx.ExecContext(ctx, `
INSERT INTO channel_entries (id, channel_key, work_item_id, enqueued_at, available_at)
VALUES ($1, 'x15', $2, $3, $4)`,
		input.id,
		"work-"+input.id,
		input.availableAt.Add(-time.Minute),
		input.availableAt,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func insertLease(t *testing.T, ctx context.Context, tx executor, input leaseInput) {
	t.Helper()

	_, err := tx.ExecContext(ctx, `
INSERT INTO leases (id, channel_entry_id, work_item_id, channel_key, granted_at, expires_at, token_digest)
VALUES ($1, $2, $3, 'x15', $4, $5, $6)`,
		input.id,
		input.entry,
		"work-"+input.entry,
		input.expiresAt.Add(-10*time.Minute),
		input.expiresAt,
		"GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14",
	)
	if err != nil {
		t.Fatal(err)
	}
}
