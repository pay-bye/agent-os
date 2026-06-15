package journal

import (
	"context"
	"testing"

	eventlog "github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestReplayReturnsEventsInOrder(t *testing.T) {
	store := &Store{queryRows: func(context.Context, string, ...any) (rowsScanner, error) {
		return &rowsValues{rows: [][]any{
			{"x20", "work_item", "x08", "x20", instant(0), int64(1), []byte(`{"event":"x48"}`)},
			{"x21", "work_item", "x08", "x21", instant(1), int64(2), []byte(`{"event":"x49"}`)},
		}}, nil
	}}

	events, err := store.Replay(context.Background(), workitem.ID("x08"))
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2", len(events))
	}
}

func TestAppendReturnsRecordedEvent(t *testing.T) {
	calls := 0
	store := &Store{query: func(context.Context, string, ...any) rowScanner {
		calls++
		if calls == 1 {
			return rowValues{values: []any{1}}
		}
		return rowValues{values: []any{int64(3)}}
	}}

	event, err := store.Append(context.Background(), eventlog.EventInput{
		ID:         eventlog.EventID("x20"),
		Coordinate: eventlog.WorkItemCoordinate(workitem.ID("x08")),
		Kind:       registry.JournalEventKindKey("x20"),
		AppendedAt: instant(0),
		Payload:    []byte(`{"event":"x48"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	if event.AppendIndex() != 3 {
		t.Fatalf("append index = %d, want 3", event.AppendIndex())
	}
}

func TestAppendReturnsUnknownEventKind(t *testing.T) {
	store := &Store{query: func(context.Context, string, ...any) rowScanner {
		return missingRow{}
	}}

	_, err := store.Append(context.Background(), eventlog.EventInput{
		ID:         eventlog.EventID("x20"),
		Coordinate: eventlog.WorkItemCoordinate(workitem.ID("x08")),
		Kind:       registry.JournalEventKindKey("x66"),
		AppendedAt: instant(0),
		Payload:    []byte(`{"event":"x48"}`),
	})

	if !registry.IsNotFound(err) {
		t.Fatalf("expected registry not-found error, got %v", err)
	}
}

func TestScanEventMapsRecordedEvent(t *testing.T) {
	event, err := ScanEvent(rowValues{values: []any{
		"x20",
		"work_item",
		"x08",
		"x20",
		instant(0),
		int64(5),
		[]byte(`{"event":"x48"}`),
	}})
	if err != nil {
		t.Fatal(err)
	}

	if event.AppendIndex() != 5 {
		t.Fatalf("append index = %d, want 5", event.AppendIndex())
	}
}
