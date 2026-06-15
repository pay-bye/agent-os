//go:build integration

package fixture

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/storage/postgres"
	"github.com/pay-bye/agent-os/internal/workitem"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

func MigratedSchema(t *testing.T, ctx context.Context) (*sql.DB, string) {
	t.Helper()

	db := postgresfixture.Open(t)
	schema := postgresfixture.CreateSchema(t, ctx, db, "x78")
	tx := postgresfixture.Begin(t, ctx, db)
	postgresfixture.SetSearchPath(t, ctx, tx, schema)
	postgresfixture.ApplyMigrations(t, ctx, tx)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return db, schema
}

func CommandsFor(db *sql.DB, schema string, ids ...string) kernel.Commands {
	return CommandsAt(db, schema, Instant(0), ids...)
}

func CommandsAt(db *sql.DB, schema string, now time.Time, ids ...string) kernel.Commands {
	return kernel.NewCommands(
		postgres.NewKernel(db, postgres.WithSearchPath(schema)),
		fixedClock{now: now},
		&sequenceIDs{values: ids},
	)
}

func SubmissionInput() kernel.SubmitInput {
	return kernel.SubmitInput{
		ID:      workitem.ID("x08"),
		Kind:    registry.ItemKindKey("x08"),
		Payload: []byte(`{"value":"x75"}`),
		DeclaredNeeds: []workitem.DeclaredNeedInput{
			{Kind: registry.NeedKindKey("x12"), Payload: []byte(`{"value":"x76"}`)},
		},
	}
}

func SubmitRoutedItem(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
	t.Helper()

	commands := CommandsFor(db, schema, "x25", "x26", "x28", "x32")
	_, err := commands.Submit(ctx, SubmissionInput())
	if err != nil {
		t.Fatal(err)
	}
}

func SubmitTwoNeeds(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
	t.Helper()

	commands := CommandsFor(db, schema, "x25", "x26", "x71", "x28", "x32")
	_, err := commands.Submit(ctx, kernel.SubmitInput{
		ID:      workitem.ID("x08"),
		Kind:    registry.ItemKindKey("x08"),
		Payload: []byte(`{"value":"x75"}`),
		DeclaredNeeds: []workitem.DeclaredNeedInput{
			{Kind: registry.NeedKindKey("x12"), Payload: []byte(`{"value":"x48"}`)},
			{Kind: registry.NeedKindKey("x13"), Payload: []byte(`{"value":"x49"}`)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func ClaimAlpha(t *testing.T, ctx context.Context, db *sql.DB, schema string) channel.Token {
	t.Helper()

	commands := CommandsFor(db, schema)
	result, err := commands.Claim(ctx, kernel.ClaimInput{
		Channel:       registry.ChannelKey("x15"),
		Lease:         channel.LeaseID("x16"),
		LeaseDuration: 10 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.Token
}

func InsertVocabulary(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
	t.Helper()

	_, err := db.ExecContext(ctx, `
SET search_path TO `+schema+`;
INSERT INTO schema_documents (key, document) VALUES ('x01', '{"title":"x91"}');
INSERT INTO item_kinds (key, schema_key, description) VALUES ('x08', 'x01', 'x91');
INSERT INTO need_kinds (key, schema_key, description) VALUES ('x12', 'x01', 'x91');
INSERT INTO need_kinds (key, schema_key, description) VALUES ('x13', 'x01', 'x92');
INSERT INTO nodes (key, description) VALUES ('x17', 'x91');
INSERT INTO nodes (key, description) VALUES ('x18', 'x92');
INSERT INTO channels (key, node_key, description) VALUES ('x15', 'x17', 'x91');
INSERT INTO channels (key, node_key, description) VALUES ('x68', 'x18', 'x92');
INSERT INTO node_capabilities (node_key, need_kind_key) VALUES ('x17', 'x12');
INSERT INTO node_capabilities (node_key, need_kind_key) VALUES ('x18', 'x13');
INSERT INTO routing_rules (need_kind_key, node_key, rule_order) VALUES ('x12', 'x17', 1);
INSERT INTO routing_rules (need_kind_key, node_key, rule_order) VALUES ('x13', 'x18', 1);
`)
	if err != nil {
		t.Fatal(err)
	}
}

func InsertConflictingEntry(t *testing.T, ctx context.Context, db *sql.DB, schema string, id string) {
	t.Helper()

	_, err := db.ExecContext(ctx, `
INSERT INTO `+schema+`.channel_entries (id, channel_key, work_item_id, enqueued_at, available_at)
VALUES ($1, 'x15', 'x11', $2, $2)`, id, Instant(0))
	if err != nil {
		t.Fatal(err)
	}
}

func RequireScalar(t *testing.T, ctx context.Context, db *sql.DB, schema string, query string, want int64) {
	t.Helper()

	var got int64
	if err := db.QueryRowContext(ctx, "SELECT set_config('search_path', $1, false)", schema).Scan(new(string)); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, query).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s = %d, want %d", query, got, want)
	}
}

func RequireNoSubmittedWork(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
	t.Helper()

	RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM work_items WHERE id = 'x08'`, int64(0))
	RequireScalar(t, ctx, db, schema, `
SELECT count(*)
FROM journal_events
WHERE coordinate_kind = 'work_item'
  AND coordinate_key = 'x08'`, int64(0))
	RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM channel_entries WHERE work_item_id = 'x08'`, int64(0))
}

func EqualStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceIDs struct {
	values []string
}

func (s *sequenceIDs) Next() string {
	if len(s.values) == 0 {
		return "x35"
	}
	value := s.values[0]
	s.values = s.values[1:]
	return value
}

func Instant(offset int) time.Time {
	return time.Date(2026, 5, 18, 12, offset, 0, 0, time.UTC)
}
