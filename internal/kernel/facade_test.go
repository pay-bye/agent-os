package kernel

import (
	"context"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestSubmitUsesInjectedClockAndIdentifiers(t *testing.T) {
	store := &recordingStore{}
	commands := NewCommands(store, fixedClock{now: instant(0)}, &sequenceIDs{
		values: []string{"x50", "x27", "x30", "x34"},
	})

	_, err := commands.Submit(context.Background(), SubmitInput{
		ID:      workitem.ID("x08"),
		Kind:    registry.ItemKindKey("x08"),
		Payload: []byte(`{"value":"x48"}`),
		DeclaredNeeds: []workitem.DeclaredNeedInput{
			{Kind: registry.NeedKindKey("x12"), Payload: []byte(`{"value":"x76"}`)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	command := store.submit
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

func TestClaimUsesLeaseWindow(t *testing.T) {
	store := &recordingStore{}
	commands := newCommandsWithTokenGenerator(
		store,
		fixedClock{now: instant(0)},
		&sequenceIDs{},
		fixedTokens{value: channel.Token("x-token")},
	)

	_, err := commands.Claim(context.Background(), ClaimInput{
		Channel:       registry.ChannelKey("x15"),
		Lease:         channel.LeaseID("x16"),
		LeaseDuration: 10 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}

	if store.claim.Lease.GrantedAt != instant(0) {
		t.Fatalf("granted at = %s, want %s", store.claim.Lease.GrantedAt, instant(0))
	}
	if store.claim.Lease.TokenDigest != channel.Digest("GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14") {
		t.Fatalf("lease token digest = %q", store.claim.Lease.TokenDigest)
	}
	if !store.claim.Lease.ExpiresAt.Equal(instant(0).Add(10 * time.Minute)) {
		t.Fatalf("expires at = %s", store.claim.Lease.ExpiresAt)
	}
}

func TestPauseUsesClockAndIdentifier(t *testing.T) {
	store := &recordingStore{}
	commands := NewCommands(store, fixedClock{now: instant(0)}, &sequenceIDs{values: []string{"x45"}})

	_, err := commands.Pause(context.Background(), PauseInput{Node: registry.NodeKey("x17")})
	if err != nil {
		t.Fatal(err)
	}

	if store.pause.Node != registry.NodeKey("x17") {
		t.Fatalf("pause node = %q, want x17", store.pause.Node)
	}
	if store.pause.Event != journal.EventID("x45") {
		t.Fatalf("pause event = %q, want x45", store.pause.Event)
	}
	if store.pause.PausedAt != instant(0) {
		t.Fatalf("paused at = %s, want %s", store.pause.PausedAt, instant(0))
	}
}

func TestRouteOutstandingInstructionUsesSeparateEventsAndEntries(t *testing.T) {
	store := &recordingStore{}
	commands := NewCommands(store, fixedClock{now: instant(0)}, &sequenceIDs{
		values: []string{"x80", "x81", "x82", "x83", "x31", "x32"},
	})

	_, err := commands.RouteOutstandingInstruction(context.Background(), ItemsInstructionInput{
		ID:        InstructionID("x70"),
		WorkItems: []workitem.ID{"x08", "x09"},
	})
	if err != nil {
		t.Fatal(err)
	}

	command := store.routeOutstandingInstruction
	if command.Record.ID != InstructionID("x70") {
		t.Fatalf("instruction id = %q, want x70", command.Record.ID)
	}
	if command.Record.Kind != "route_outstanding" {
		t.Fatalf("instruction kind = %q, want route_outstanding", command.Record.Kind)
	}
	if command.Record.RecordedAt != instant(0) {
		t.Fatalf("recorded at = %s, want %s", command.Record.RecordedAt, instant(0))
	}
	if command.Events[0] != journal.EventID("x80") || command.Events[1] != journal.EventID("x81") {
		t.Fatalf("events = %v, want x80/x81", command.Events)
	}
	if command.RouteEvents[0] != journal.EventID("x82") || command.RouteEvents[1] != journal.EventID("x83") {
		t.Fatalf("route events = %v, want x82/x83", command.RouteEvents)
	}
	if command.Entries[0] != channel.EntryID("x31") || command.Entries[1] != channel.EntryID("x32") {
		t.Fatalf("entries = %v, want x31/x32", command.Entries)
	}
}
