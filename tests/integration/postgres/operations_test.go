//go:build integration

package postgres_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	readpostgres "github.com/pay-bye/agent-os/internal/readmodel/postgres"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

func TestOperationsDerivesStoragePressureAndJournalWindow(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	postgresfixture.SetSearchPath(t, ctx, tx, "pg_temp")
	insertChannel(t, ctx, tx)
	insertEventKind(t, ctx, tx)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	insertEntry(t, ctx, tx, entryInput{id: "x01", availableAt: now.Add(-2 * time.Minute)})
	insertEntry(t, ctx, tx, entryInput{id: "x02", availableAt: now.Add(time.Minute)})
	insertEntry(t, ctx, tx, entryInput{id: "x03", availableAt: now.Add(-time.Minute)})
	insertEntry(t, ctx, tx, entryInput{id: "x04", availableAt: now.Add(-time.Minute)})
	insertLease(t, ctx, tx, leaseInput{id: "x13", entry: "x03", expiresAt: now.Add(time.Minute)})
	insertLease(t, ctx, tx, leaseInput{id: "x14", entry: "x04", expiresAt: now.Add(-time.Minute)})
	insertJournal(t, ctx, tx, "x21", now.Add(-10*time.Second))
	insertJournal(t, ctx, tx, "x22", now.Add(-2*time.Minute))
	insertJournal(t, ctx, tx, "x23", now.Add(-6*time.Minute))

	reader := readpostgres.NewOperations(tx)
	pressure, err := reader.Pressure(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	window, err := reader.Journal(ctx, now, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	if pressure.Depth != 4 {
		t.Fatalf("depth = %d, want 4", pressure.Depth)
	}
	if pressure.Available != 2 {
		t.Fatalf("available = %d, want 2", pressure.Available)
	}
	if pressure.Held != 1 {
		t.Fatalf("held = %d, want 1", pressure.Held)
	}
	if pressure.Expired != 1 {
		t.Fatalf("expired = %d, want 1", pressure.Expired)
	}
	if pressure.OldestAvailableAgeSeconds != 120 {
		t.Fatalf("oldest age = %d, want 120", pressure.OldestAvailableAgeSeconds)
	}
	if window.Appends != 2 {
		t.Fatalf("journal appends = %d, want 2", window.Appends)
	}
	if window.WindowSeconds != 300 {
		t.Fatalf("window seconds = %d, want 300", window.WindowSeconds)
	}
}

func TestOperationsDerivesNodeRosterFromRegistryAndExclusions(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	postgresfixture.SetSearchPath(t, ctx, tx, "pg_temp")
	insertRoster(t, ctx, tx)

	nodes, err := readpostgres.NewOperations(tx).Nodes(ctx, readpostgres.NodeQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) != 2 {
		t.Fatalf("nodes = %d, want 2", len(nodes))
	}
	if nodes[0].Key != "x17" || nodes[0].Channel != "x15" || nodes[0].Routable {
		t.Fatalf("first node = %+v, want excluded x17", nodes[0])
	}
	if nodes[1].Key != "x18" || nodes[1].Channel != "x68" || !nodes[1].Routable {
		t.Fatalf("second node = %+v, want routable x18", nodes[1])
	}
	if !equalStrings(nodes[1].NeedKinds, []string{"x12", "x13"}) {
		t.Fatalf("need kinds = %v, want x12/x13", nodes[1].NeedKinds)
	}
}

func TestOperationsDerivesChannelLocatorRows(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	postgresfixture.SetSearchPath(t, ctx, tx, "pg_temp")
	insertRoster(t, ctx, tx)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	insertEntryInChannel(t, ctx, tx, channelEntryInput{id: "x01", channel: "x15", availableAt: now.Add(-2 * time.Minute)})
	insertEntryInChannel(t, ctx, tx, channelEntryInput{id: "x02", channel: "x15", availableAt: now.Add(time.Minute)})
	insertEntryInChannel(t, ctx, tx, channelEntryInput{id: "x03", channel: "x68", availableAt: now.Add(-30 * time.Second)})
	insertLease(t, ctx, tx, leaseInput{id: "x13", entry: "x03", expiresAt: now.Add(time.Minute)})

	channels, err := readpostgres.NewOperations(tx).Channels(ctx, now, readpostgres.ChannelQuery{
		Limit:            10,
		OlderThanSeconds: 60,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(channels) != 1 {
		t.Fatalf("channels = %+v, want one older available channel", channels)
	}
	got := channels[0]
	if got.Key != "x15" || got.Node != "x17" {
		t.Fatalf("channel identity = %+v, want x15/x17", got)
	}
	if got.Depth != 2 || got.Available != 1 || got.OldestAvailableAgeSeconds != 120 {
		t.Fatalf("channel pressure = %+v, want depth 2 available 1 age 120", got)
	}
}

func TestOperationsDerivesChannelItemsWithoutLeaseTokens(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	postgresfixture.SetSearchPath(t, ctx, tx, "pg_temp")
	insertRoster(t, ctx, tx)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	insertEntryInChannel(t, ctx, tx, channelEntryInput{id: "x01", channel: "x15", availableAt: now.Add(-2 * time.Minute)})
	insertEntryInChannel(t, ctx, tx, channelEntryInput{id: "x02", channel: "x15", availableAt: now.Add(-time.Minute)})
	insertLease(t, ctx, tx, leaseInput{id: "x13", entry: "x01", expiresAt: now.Add(time.Minute)})
	insertLease(t, ctx, tx, leaseInput{id: "x14", entry: "x02", expiresAt: now.Add(-time.Minute)})

	items, err := readpostgres.NewOperations(tx).ChannelItems(ctx, now, readpostgres.ChannelItemQuery{
		Channel: "x15",
		Limit:   10,
		Lease:   readpostgres.LeaseViewHeld,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatalf("items = %+v, want held item only", items)
	}
	got := items[0]
	if got.Entry != "x01" || got.WorkItem != "work-x01" || got.Node != "x17" {
		t.Fatalf("item identity = %+v, want held x01", got)
	}
	if got.Lease == nil || got.Lease.ID != "x13" {
		t.Fatalf("lease = %+v, want redacted x13 lease", got.Lease)
	}
	body, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if containsAny(string(body), "token_digest", "lease_token") {
		t.Fatalf("item leaked token material: %s", body)
	}
}

func TestOperationsDerivesItemDetailWithoutPayload(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	postgresfixture.SetSearchPath(t, ctx, tx, "pg_temp")
	insertRoster(t, ctx, tx)
	insertItem(t, ctx, tx, "x08")
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	insertEntryInChannel(t, ctx, tx, channelEntryInput{id: "x01", channel: "x15", workItem: "x08", availableAt: now.Add(-time.Minute)})
	insertLeaseForWork(t, ctx, tx, leaseInput{id: "x13", entry: "x01", expiresAt: now.Add(time.Minute)}, "x08")
	insertDeclaredNeed(t, ctx, tx, "x21", "x08", now.Add(-30*time.Second))

	got, err := readpostgres.NewOperations(tx).Item(ctx, now, "x08")
	if err != nil {
		t.Fatal(err)
	}

	if got.WorkItem != "x08" || got.Kind != "x03" {
		t.Fatalf("item = %+v, want x08/x03", got)
	}
	if got.Entry == nil || got.Entry.Entry != "x01" || got.Entry.AgeSeconds != 60 {
		t.Fatalf("entry = %+v, want x01 age 60", got.Entry)
	}
	if got.Lease == nil || got.Lease.ID != "x13" {
		t.Fatalf("lease = %+v, want x13", got.Lease)
	}
	if got.Need == nil || got.Need.Kind != "x12" || got.Need.Target != "x17" {
		t.Fatalf("need = %+v, want x12 target x17", got.Need)
	}
	body, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if containsAny(string(body), "payload", "token_digest", "lease_token") {
		t.Fatalf("item detail leaked forbidden material: %s", body)
	}
}

func TestOperationsReplaysJournalMetadataAllowlist(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	postgresfixture.SetSearchPath(t, ctx, tx, "pg_temp")
	insertEventKind(t, ctx, tx)
	appendedAt := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	_, err := tx.ExecContext(ctx, `
INSERT INTO journal_events (id, coordinate_kind, coordinate_key, event_kind_key, appended_at, payload)
VALUES (
  'x21',
  'work_item',
  'x08',
  'x31',
  $1,
  '{"work_item_id":"x08","need_kind":"x12","payload":{"secret":"x91"},"failure_payload":{"secret":"x92"},"token_digest":"x93","operator_key":"x94"}'
)`, appendedAt)
	if err != nil {
		t.Fatal(err)
	}

	events, err := readpostgres.NewOperations(tx).ItemJournal(ctx, readpostgres.JournalQuery{
		WorkItem: "x08",
		Limit:    10,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 1 {
		t.Fatalf("events = %+v, want one event", events)
	}
	metadata := events[0].Metadata
	if metadata["work_item_id"] != "x08" || metadata["need_kind"] != "x12" {
		t.Fatalf("metadata = %+v, want allowed scalar coordinates", metadata)
	}
	if _, ok := metadata["payload"]; ok {
		t.Fatalf("metadata leaked payload: %+v", metadata)
	}
	if _, ok := metadata["token_digest"]; ok {
		t.Fatalf("metadata leaked token digest: %+v", metadata)
	}
}

func insertEventKind(t *testing.T, ctx context.Context, tx executor) {
	t.Helper()

	_, err := tx.ExecContext(ctx, `
INSERT INTO journal_event_kinds (key, schema_key, description)
VALUES ('x31', NULL, 'x91')`)
	if err != nil {
		t.Fatal(err)
	}
}

func insertRoster(t *testing.T, ctx context.Context, tx executor) {
	t.Helper()

	_, err := tx.ExecContext(ctx, `
INSERT INTO schema_documents (key, document) VALUES ('x91', '{"title":"x91"}');
INSERT INTO need_kinds (key, schema_key, description) VALUES ('x12', 'x91', 'x91');
INSERT INTO need_kinds (key, schema_key, description) VALUES ('x13', 'x91', 'x92');
INSERT INTO nodes (key, description) VALUES ('x17', 'x91');
INSERT INTO nodes (key, description) VALUES ('x18', 'x92');
INSERT INTO channels (key, node_key, description) VALUES ('x15', 'x17', 'x91');
INSERT INTO channels (key, node_key, description) VALUES ('x68', 'x18', 'x92');
INSERT INTO node_capabilities (node_key, need_kind_key) VALUES ('x17', 'x12');
INSERT INTO node_capabilities (node_key, need_kind_key) VALUES ('x18', 'x12');
INSERT INTO node_capabilities (node_key, need_kind_key) VALUES ('x18', 'x13');
INSERT INTO routing_exclusions (node_key) VALUES ('x17');
`)
	if err != nil {
		t.Fatal(err)
	}
}

type channelEntryInput struct {
	id          string
	channel     string
	workItem    string
	availableAt time.Time
}

func insertEntryInChannel(t *testing.T, ctx context.Context, tx executor, input channelEntryInput) {
	t.Helper()

	workItem := input.workItem
	if workItem == "" {
		workItem = "work-" + input.id
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO channel_entries (id, channel_key, work_item_id, enqueued_at, available_at)
VALUES ($1, $2, $3, $4, $5)`,
		input.id,
		input.channel,
		workItem,
		input.availableAt.Add(-time.Minute),
		input.availableAt,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func insertLeaseForWork(t *testing.T, ctx context.Context, tx executor, input leaseInput, workItem string) {
	t.Helper()

	_, err := tx.ExecContext(ctx, `
INSERT INTO leases (id, channel_entry_id, work_item_id, channel_key, granted_at, expires_at, token_digest)
VALUES ($1, $2, $3, 'x15', $4, $5, $6)`,
		input.id,
		input.entry,
		workItem,
		input.expiresAt.Add(-10*time.Minute),
		input.expiresAt,
		"GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14",
	)
	if err != nil {
		t.Fatal(err)
	}
}

func insertItem(t *testing.T, ctx context.Context, tx executor, id string) {
	t.Helper()

	_, err := tx.ExecContext(ctx, `
	INSERT INTO item_kinds (key, schema_key, description)
	VALUES ('x03', 'x91', 'x91')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.ExecContext(ctx, `
	INSERT INTO work_items (id, item_kind_key, payload, submitted_at)
	VALUES ($1, 'x03', '{"secret":"x91"}', '2026-05-18T11:55:00Z')`, id)
	if err != nil {
		t.Fatal(err)
	}
}

func insertDeclaredNeed(t *testing.T, ctx context.Context, tx executor, event string, workItem string, appendedAt time.Time) {
	t.Helper()

	_, err := tx.ExecContext(ctx, `
INSERT INTO journal_events (id, coordinate_kind, coordinate_key, event_kind_key, appended_at, payload)
VALUES ($1, 'work_item', $2, 'x41', $3, '{"need_kind":"x12","target_node":"x17","payload":{"secret":"x91"}}')`,
		event,
		workItem,
		appendedAt,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func containsAny(body string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(body, value) {
			return true
		}
	}
	return false
}

func equalStrings(left []string, right []string) bool {
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

func insertJournal(t *testing.T, ctx context.Context, tx executor, id string, appendedAt time.Time) {
	t.Helper()

	_, err := tx.ExecContext(ctx, `
INSERT INTO journal_events (id, coordinate_kind, coordinate_key, event_kind_key, appended_at, payload)
	VALUES ($1, 'work_item', $2, 'x31', $3, '{"value":"x91"}')`,
		id,
		"x41-"+id,
		appendedAt,
	)
	if err != nil {
		t.Fatal(err)
	}
}
