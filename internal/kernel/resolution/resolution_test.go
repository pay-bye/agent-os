package resolution

import (
	"errors"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestNewBuildsResolutionEventsAndCopiesFailurePayload(t *testing.T) {
	payload := []byte(`{"error":"x48"}`)
	command, err := New(Input{
		Lease:          "x16",
		Token:          "x-token",
		FailurePayload: payload,
	}, instant(0), &sequenceIDs{values: []string{"x31", "x53", "x54"}})
	if err != nil {
		t.Fatal(err)
	}
	payload[0] = '['

	if command.Event != journal.EventID("x31") {
		t.Fatalf("event = %q, want x31", command.Event)
	}
	if command.RouteEvent != journal.EventID("x53") {
		t.Fatalf("route event = %q, want x53", command.RouteEvent)
	}
	if command.Entry != channel.EntryID("x54") {
		t.Fatalf("entry = %q, want x54", command.Entry)
	}
	if command.TokenDigest != channel.Digest("GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14") {
		t.Fatalf("token digest = %q", command.TokenDigest)
	}
	if string(command.FailurePayload) != `{"error":"x48"}` {
		t.Fatalf("failure payload = %s", command.FailurePayload)
	}
}

func TestNewBuildsDeclaredNeedEvents(t *testing.T) {
	command, err := New(Input{
		Lease: "x16",
		Token: "x-token",
		DeclaredNeeds: []workitem.DeclaredNeedInput{
			{Kind: registry.NeedKindKey("x12"), Payload: []byte(`{"value":"x76"}`)},
		},
	}, instant(0), &sequenceIDs{values: []string{"x31", "x52", "x53", "x54"}})
	if err != nil {
		t.Fatal(err)
	}

	if command.NeedEvents[0] != journal.EventID("x52") {
		t.Fatalf("need event = %q, want x52", command.NeedEvents[0])
	}
	if len(command.DeclaredNeeds) != 1 {
		t.Fatalf("declared needs = %d, want 1", len(command.DeclaredNeeds))
	}
	if command.FailurePayload != nil {
		t.Fatalf("failure payload = %v, want nil", command.FailurePayload)
	}
}

func TestNewRejectsBlankToken(t *testing.T) {
	_, err := New(Input{Token: " "}, instant(0), &sequenceIDs{})

	if !errors.Is(err, channel.ErrEmptyToken) {
		t.Fatalf("error = %v, want empty token", err)
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
