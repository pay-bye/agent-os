//go:build integration

package fixture

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

type Facts struct {
	ChannelEntryFacts string
	JournalFacts      string
	LeaseFacts        string
	WorkItemFacts     string
}

type factReader struct {
	t      *testing.T
	ctx    context.Context
	db     *sql.DB
	schema string
}

func ReadFacts(t *testing.T, ctx context.Context, db *sql.DB, schema string) Facts {
	t.Helper()

	reader := factReader{t: t, ctx: ctx, db: db, schema: schema}
	return reader.read()
}

func RegistryFacts(t *testing.T, ctx context.Context, db *sql.DB, schema string) string {
	t.Helper()

	reader := factReader{t: t, ctx: ctx, db: db, schema: schema}
	return reader.registryFacts()
}

func (reader factReader) read() Facts {
	return Facts{
		ChannelEntryFacts: reader.channelEntryFacts(),
		JournalFacts:      reader.journalFacts(),
		LeaseFacts:        reader.leaseFacts(),
		WorkItemFacts:     reader.workItemFacts(),
	}
}

func RequireUnchangedFacts(t *testing.T, before Facts, after Facts) {
	t.Helper()

	if after.JournalFacts != before.JournalFacts {
		t.Fatalf("journal facts = %q, want %q", after.JournalFacts, before.JournalFacts)
	}
	if after.ChannelEntryFacts != before.ChannelEntryFacts {
		t.Fatalf("channel entry facts = %q, want %q", after.ChannelEntryFacts, before.ChannelEntryFacts)
	}
	if after.LeaseFacts != before.LeaseFacts {
		t.Fatalf("lease facts = %q, want %q", after.LeaseFacts, before.LeaseFacts)
	}
	if after.WorkItemFacts != before.WorkItemFacts {
		t.Fatalf("work item facts = %q, want %q", after.WorkItemFacts, before.WorkItemFacts)
	}
}

func (reader factReader) channelEntryFacts() string {
	return reader.text(`
SELECT COALESCE(
  string_agg(
    concat_ws('|', id, channel_key, work_item_id, enqueued_at::text, available_at::text),
    E'\n' ORDER BY id
  ),
  ''
)
FROM ` + reader.schema + `.channel_entries`)
}

func (reader factReader) journalFacts() string {
	return reader.text(`
SELECT COALESCE(
  string_agg(
    concat_ws('|', id, coordinate_kind, coordinate_key, event_kind_key, appended_at::text, append_index::text, payload::text),
    E'\n' ORDER BY append_index, id
  ),
  ''
)
FROM ` + reader.schema + `.journal_events`)
}

func (reader factReader) leaseFacts() string {
	return reader.text(`
SELECT COALESCE(
  string_agg(
    concat_ws(
      '|',
      id,
      channel_entry_id,
      work_item_id,
      channel_key,
      granted_at::text,
      expires_at::text,
      token_digest
    ),
    E'\n' ORDER BY id
  ),
  ''
)
FROM ` + reader.schema + `.leases`)
}

func (reader factReader) workItemFacts() string {
	return reader.text(`
SELECT COALESCE(
  string_agg(
    concat_ws('|', id, item_kind_key, payload::text, submitted_at::text),
    E'\n' ORDER BY id
  ),
  ''
)
FROM ` + reader.schema + `.work_items`)
}

func (reader factReader) registryFacts() string {
	return reader.text(`
SELECT concat_ws(E'\n',
  (SELECT COALESCE(string_agg(concat_ws('|', key, description), E'\n' ORDER BY key), '') FROM ` + reader.schema + `.nodes),
  (SELECT COALESCE(string_agg(concat_ws('|', key, node_key, description), E'\n' ORDER BY key), '') FROM ` + reader.schema + `.channels),
  (SELECT COALESCE(string_agg(concat_ws('|', node_key, need_kind_key), E'\n' ORDER BY node_key, need_kind_key), '') FROM ` + reader.schema + `.node_capabilities),
  (SELECT COALESCE(string_agg(concat_ws('|', need_kind_key, node_key, rule_order::text), E'\n' ORDER BY need_kind_key, rule_order), '') FROM ` + reader.schema + `.routing_rules)
)`)
}

func (reader factReader) text(query string) string {
	reader.t.Helper()

	var facts string
	err := reader.db.QueryRowContext(reader.ctx, query).Scan(&facts)
	if err != nil {
		reader.t.Fatal(err)
	}
	return facts
}

func RequireLeaseExpiry(
	t *testing.T,
	ctx context.Context,
	db *sql.DB,
	schema string,
	id string,
	want time.Time,
) {
	t.Helper()

	got := leaseExpiry(t, ctx, db, schema, id)
	if !got.Equal(want) {
		t.Fatalf("lease expiry = %s, want %s", got, want)
	}
}

func leaseExpiry(t *testing.T, ctx context.Context, db *sql.DB, schema string, id string) time.Time {
	t.Helper()

	var expiresAt time.Time
	err := db.QueryRowContext(ctx, `
SELECT expires_at
FROM `+schema+`.leases
WHERE id = $1`, id).Scan(&expiresAt)
	if err != nil {
		t.Fatal(err)
	}
	return expiresAt
}
