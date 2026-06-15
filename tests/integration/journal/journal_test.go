//go:build integration

package journal_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/registry"
	journalstore "github.com/pay-bye/agent-os/internal/storage/postgres/journal"
	"github.com/pay-bye/agent-os/internal/workitem"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

func TestAppendReplaysEventsInAppendOrder(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	insertVocabulary(t, ctx, tx)
	store := journalstore.New(tx)
	appendEvent(t, ctx, store, "x20", "x08", "x20", `{"event":"x48"}`)
	appendEvent(t, ctx, store, "x69", "x70", "x20", `{"event":"other"}`)
	appendEvent(t, ctx, store, "x21", "x08", "x21", `{"event":"x49"}`)

	events, err := store.Replay(ctx, workitem.ID("x08"))
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2", len(events))
	}
	requireEvent(t, events[0], "x20", 1)
	requireEvent(t, events[1], "x21", 3)
}

func TestAppendStoresNodeCoordinateAuditWithoutWorkItem(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	insertVocabulary(t, ctx, tx)
	store := journalstore.New(tx)

	event, err := store.Append(ctx, journal.EventInput{
		ID:         journal.EventID("x20"),
		Coordinate: journal.NodeCoordinate(registry.NodeKey("x17")),
		Kind:       registry.JournalEventKindKey("x20"),
		AppendedAt: instant(0),
		Payload:    []byte(`{"event":"x48"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	if event.Coordinate() != journal.NodeCoordinate(registry.NodeKey("x17")) {
		t.Fatalf("coordinate = %+v, want node x17", event.Coordinate())
	}
}

func TestAppendReplaysNodeEventsInAppendOrder(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	insertVocabulary(t, ctx, tx)
	store := journalstore.New(tx)
	appendNodeEvent(t, ctx, store, "x20", "x17", "x45", `{"node_key":"x17"}`)
	appendNodeEvent(t, ctx, store, "x21", "x18", "x45", `{"node_key":"x18"}`)
	appendNodeEvent(t, ctx, store, "x22", "x17", "x46", `{"node_key":"x17"}`)

	events, err := store.ReplayNode(ctx, registry.NodeKey("x17"))
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2", len(events))
	}
	requireEvent(t, events[0], "x20", 1)
	requireEvent(t, events[1], "x22", 3)
}

func TestAppendRejectsUnknownCoordinate(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	insertVocabulary(t, ctx, tx)
	store := journalstore.New(tx)

	_, err := store.Append(ctx, journal.EventInput{
		ID:         journal.EventID("x20"),
		Coordinate: journal.Coordinate{},
		Kind:       registry.JournalEventKindKey("x20"),
		AppendedAt: instant(0),
		Payload:    []byte(`{"event":"x48"}`),
	})

	if !errors.Is(err, journal.ErrEmptyCoordinate) {
		t.Fatalf("error = %v, want empty coordinate", err)
	}
}

func TestStorageRejectsUnknownCoordinateKind(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	insertVocabulary(t, ctx, tx)

	_, err := tx.ExecContext(ctx, `
	INSERT INTO journal_events (id, coordinate_kind, coordinate_key, event_kind_key, appended_at, payload)
	VALUES ('x20', 'workflow', 'x08', 'x20', $1, '{"event":"x48"}')`, instant(0))

	if err == nil {
		t.Fatal("unknown coordinate kind must fail")
	}
	if !strings.Contains(err.Error(), "journal_events_coordinate_kind_known") {
		t.Fatalf("error = %v, want coordinate kind constraint", err)
	}
}

func TestAppendRejectsUnknownEventKind(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	store := journalstore.New(tx)

	_, err := store.Append(ctx, eventInput("x20", "x08", "x66", `{"event":"x48"}`))

	if !registry.IsNotFound(err) {
		t.Fatalf("expected registry not-found error, got %v", err)
	}
}

func TestAppendRejectsCallerSuppliedAppendOrder(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	insertVocabulary(t, ctx, tx)

	_, err := tx.ExecContext(ctx, `
INSERT INTO journal_events (id, coordinate_kind, coordinate_key, event_kind_key, appended_at, append_index, payload)
VALUES ('x20', 'work_item', 'x08', 'x20', $1, 99, '{"event":"x48"}')`, instant(0))

	if err == nil {
		t.Fatal("caller supplied append order must fail")
	}
}

func TestAppendRejectsMalformedPayload(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	insertVocabulary(t, ctx, tx)
	store := journalstore.New(tx)

	_, err := store.Append(ctx, eventInput("x20", "x08", "x20", `{"broken"`))

	if !errors.Is(err, journal.ErrMalformedPayload) {
		t.Fatalf("error = %v, want malformed payload", err)
	}
}

func appendEvent(
	t *testing.T,
	ctx context.Context,
	store *journalstore.Store,
	id string,
	item string,
	kind string,
	payload string,
) {
	t.Helper()

	_, err := store.Append(ctx, eventInput(id, item, kind, payload))
	if err != nil {
		t.Fatal(err)
	}
}

func appendNodeEvent(
	t *testing.T,
	ctx context.Context,
	store *journalstore.Store,
	id string,
	node string,
	kind string,
	payload string,
) {
	t.Helper()

	_, err := store.Append(ctx, journal.EventInput{
		ID:         journal.EventID(id),
		Coordinate: journal.NodeCoordinate(registry.NodeKey(node)),
		Kind:       registry.JournalEventKindKey(kind),
		AppendedAt: instant(0),
		Payload:    []byte(payload),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func eventInput(id string, item string, kind string, payload string) journal.EventInput {
	return journal.EventInput{
		ID:         journal.EventID(id),
		Coordinate: journal.WorkItemCoordinate(workitem.ID(item)),
		Kind:       registry.JournalEventKindKey(kind),
		AppendedAt: instant(0),
		Payload:    []byte(payload),
	}
}

func requireEvent(t *testing.T, event journal.Event, id string, index int64) {
	t.Helper()

	if event.ID() != journal.EventID(id) {
		t.Fatalf("event identity = %q, want %s", event.ID(), id)
	}
	if event.AppendIndex() != index {
		t.Fatalf("append index = %d, want %d", event.AppendIndex(), index)
	}
}

func insertVocabulary(t *testing.T, ctx context.Context, tx *sql.Tx) {
	t.Helper()

	_, err := tx.ExecContext(ctx, `
INSERT INTO schema_documents (key, document) VALUES ('x01', '{"title":"First"}');
INSERT INTO journal_event_kinds (key, schema_key, description) VALUES ('x20', 'x01', 'First');
INSERT INTO journal_event_kinds (key, schema_key, description) VALUES ('x21', 'x01', 'Second');
INSERT INTO journal_event_kinds (key, schema_key, description) VALUES ('x22', 'x01', 'Third');
`)
	if err != nil {
		t.Fatal(err)
	}
}

func instant(offset int) time.Time {
	return time.Date(2026, 5, 18, 12, offset, 0, 0, time.UTC)
}
