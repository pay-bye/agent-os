//go:build integration

package fixture

import (
	"context"
	"database/sql"
	"testing"
)

func InsertCapability(t *testing.T, ctx context.Context, db *sql.DB, schema string, node string, need string) {
	t.Helper()

	_, err := db.ExecContext(ctx, `
INSERT INTO `+schema+`.node_capabilities (node_key, need_kind_key)
VALUES ($1, $2)`, node, need)
	if err != nil {
		t.Fatal(err)
	}
}

func InsertConflictingEvent(t *testing.T, ctx context.Context, db *sql.DB, schema string, id string) {
	t.Helper()

	_, err := db.ExecContext(ctx, `
INSERT INTO `+schema+`.journal_events (id, coordinate_kind, coordinate_key, event_kind_key, appended_at, payload)
VALUES ($1, 'work_item', 'x11', 'x40', $2, '{}')`, id, Instant(0))
	if err != nil {
		t.Fatal(err)
	}
}

func RequireEventKinds(t *testing.T, ctx context.Context, db *sql.DB, schema string, want ...string) {
	t.Helper()

	rows, err := db.QueryContext(ctx, `
SELECT event_kind_key
FROM `+schema+`.journal_events
ORDER BY append_index`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var got []string
	for rows.Next() {
		var kind string
		if err := rows.Scan(&kind); err != nil {
			t.Fatal(err)
		}
		got = append(got, kind)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !EqualStrings(got, want) {
		t.Fatalf("event kinds = %v, want %v", got, want)
	}
}
