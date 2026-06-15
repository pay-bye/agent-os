package submission

import (
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestNewUsesInjectedClockAndIdentifiers(t *testing.T) {
	command, err := New(Input{
		ID:      workitem.ID("x08"),
		Kind:    registry.ItemKindKey("x08"),
		Payload: []byte(`{"value":"x48"}`),
		DeclaredNeeds: []workitem.DeclaredNeedInput{
			{Kind: registry.NeedKindKey("x12"), Payload: []byte(`{"value":"x76"}`)},
		},
	}, instant(0), &sequenceIDs{values: []string{"x50", "x27", "x30", "x34"}})
	if err != nil {
		t.Fatal(err)
	}

	if command.SubmittedAt != instant(0) {
		t.Fatalf("submitted at = %s, want %s", command.SubmittedAt, instant(0))
	}
	if command.ItemEvent != journal.EventID("x50") {
		t.Fatalf("item event = %q, want x50", command.ItemEvent)
	}
	if command.NeedEvents[0] != journal.EventID("x27") {
		t.Fatalf("need event = %q, want x27", command.NeedEvents[0])
	}
	if command.RouteEvent != journal.EventID("x30") {
		t.Fatalf("route event = %q, want x30", command.RouteEvent)
	}
	if command.Entry != channel.EntryID("x34") {
		t.Fatalf("entry = %q, want x34", command.Entry)
	}
}

type sequenceIDs struct {
	values []string
}

func (s *sequenceIDs) Next() string {
	value := s.values[0]
	s.values = s.values[1:]
	return value
}

func instant(offset int) time.Time {
	return time.Date(2026, 5, 18, 12, offset, 0, 0, time.UTC)
}
