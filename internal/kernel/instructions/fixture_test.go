package instructions

import (
	"errors"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/registry"
)

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

func instant(offset int) time.Time {
	return time.Date(2026, 5, 18, 12, offset, 0, 0, time.UTC)
}

func recordedEvent(
	t *testing.T,
	id string,
	kind registry.JournalEventKindKey,
	payload string,
	index int64,
) journal.Event {
	t.Helper()

	event, err := journal.NewRecordedEvent(journal.EventInput{
		ID:         journal.EventID(id),
		Coordinate: journal.WorkItemCoordinate("x08"),
		Kind:       kind,
		AppendedAt: instant(0),
		Payload:    []byte(payload),
	}, index)
	if err != nil {
		t.Fatal(err)
	}
	return event
}

func recordAt(id ID) Record {
	return Record{ID: id, Kind: "test", RecordedAt: instant(2)}
}

func requireResult(t *testing.T, result Result, value ResultValue, precondition string) {
	t.Helper()

	if result.Result != value || result.FailedPrecondition != precondition {
		t.Fatalf("result = %+v, want %s/%s", result, value, precondition)
	}
}

func requireOutcome(t *testing.T, outcome Outcome, kind registry.JournalEventKindKey) {
	t.Helper()

	if outcome.Kind != kind {
		t.Fatalf("outcome kind = %s, want %s", outcome.Kind, kind)
	}
	if len(outcome.Payload) == 0 {
		t.Fatal("outcome payload is empty")
	}
}

func node(t *testing.T, key registry.NodeKey, channel registry.ChannelKey, need registry.NeedKindKey) registry.Node {
	t.Helper()

	node, err := registry.NewNode(registry.NodeInput{
		Key:          key,
		Description:  "test node",
		Channel:      channel,
		Capabilities: []registry.NeedKindKey{need},
	})
	if err != nil {
		t.Fatal(err)
	}
	return node
}

func requireError(t *testing.T, err error, want error) {
	t.Helper()

	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}
