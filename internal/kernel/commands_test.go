package kernel

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestClaimRejectsInvalidDuration(t *testing.T) {
	commands := NewCommands(&recordingStore{}, fixedClock{now: instant(0)}, &sequenceIDs{})

	_, err := commands.Claim(context.Background(), ClaimInput{
		Lease: channel.LeaseID("x16"),
	})

	if err != ErrInvalidLeaseDuration {
		t.Fatalf("error = %v, want invalid lease duration", err)
	}
}

func TestClaimStoresDigestAndReturnsToken(t *testing.T) {
	lease := mustLease(t, "x16")
	store := &recordingStore{claimResult: ClaimResult{
		Lease:    lease,
		WorkItem: lease.WorkItem(),
	}}
	commands := newCommandsWithTokenGenerator(
		store,
		fixedClock{now: instant(0)},
		&sequenceIDs{},
		fixedTokens{value: channel.Token("x-token")},
	)

	result, err := commands.Claim(context.Background(), ClaimInput{
		Channel:       registry.ChannelKey("x15"),
		Lease:         channel.LeaseID("x16"),
		LeaseDuration: 10 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}

	if store.claim.Lease.TokenDigest != channel.Digest("GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14") {
		t.Fatalf("token digest = %q", store.claim.Lease.TokenDigest)
	}
	if result.Token != channel.Token("x-token") {
		t.Fatalf("token = %q, want x-token", result.Token)
	}
}

func TestClaimGenerationFailureDoesNotCallStore(t *testing.T) {
	want := errors.New("token unavailable")
	store := &recordingStore{}
	commands := newCommandsWithTokenGenerator(
		store,
		fixedClock{now: instant(0)},
		&sequenceIDs{},
		fixedTokens{err: want},
	)

	_, err := commands.Claim(context.Background(), ClaimInput{
		Channel:       registry.ChannelKey("x15"),
		Lease:         channel.LeaseID("x16"),
		LeaseDuration: time.Minute,
	})

	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want token error", err)
	}
	if store.claimCalled {
		t.Fatal("store claim was called")
	}
}

func TestAckAndNackUseSeparateStoreMethods(t *testing.T) {
	store := &recordingStore{}
	commands := NewCommands(store, fixedClock{now: instant(0)}, &sequenceIDs{
		values: []string{"x31", "x51", "x52", "x53", "x54", "x55"},
	})

	if _, err := commands.Ack(context.Background(), ResolutionInput{
		Lease: channel.LeaseID("x16"),
		Token: channel.Token("x-token"),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := commands.Nack(context.Background(), ResolutionInput{
		Lease: channel.LeaseID("x56"),
		Token: channel.Token("x-token"),
	}); err != nil {
		t.Fatal(err)
	}

	if store.ack.Event != journal.EventID("x31") {
		t.Fatalf("ack event = %q, want x31", store.ack.Event)
	}
	if store.nack.Event != journal.EventID("x53") {
		t.Fatalf("nack event = %q, want x53", store.nack.Event)
	}
	if store.ack.TokenDigest != channel.Digest("GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14") {
		t.Fatalf("ack token digest = %q", store.ack.TokenDigest)
	}
	if store.nack.TokenDigest != channel.Digest("GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14") {
		t.Fatalf("nack token digest = %q", store.nack.TokenDigest)
	}
}

func TestExtendAndHeartbeatUseClock(t *testing.T) {
	store := &recordingStore{}
	commands := NewCommands(store, fixedClock{now: instant(0)}, &sequenceIDs{})

	if _, err := commands.Extend(context.Background(), ExtendInput{
		Lease:     channel.LeaseID("x16"),
		Token:     channel.Token("x-token"),
		ExpiresAt: instant(10),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := commands.Heartbeat(context.Background(), HeartbeatInput{
		Lease: channel.LeaseID("x16"),
		Token: channel.Token("x-token"),
	}); err != nil {
		t.Fatal(err)
	}

	if store.extend.CheckedAt != instant(0) {
		t.Fatalf("extend checked at = %s, want %s", store.extend.CheckedAt, instant(0))
	}
	if store.heartbeat.CheckedAt != instant(0) {
		t.Fatalf("heartbeat checked at = %s, want %s", store.heartbeat.CheckedAt, instant(0))
	}
	if store.extend.TokenDigest != channel.Digest("GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14") {
		t.Fatalf("extend token digest = %q", store.extend.TokenDigest)
	}
	if store.heartbeat.TokenDigest != channel.Digest("GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14") {
		t.Fatalf("heartbeat token digest = %q", store.heartbeat.TokenDigest)
	}
}

func TestInstructionFacadeDelegatesToStore(t *testing.T) {
	store := &recordingStore{}
	commands := NewCommands(store, fixedClock{now: instant(0)}, &sequenceIDs{
		values: []string{"x80", "x81", "x82", "x83", "x84", "x85", "x86", "x87", "x31"},
	})

	calls := []struct {
		name string
		call func() error
		want func(*testing.T)
	}{
		{
			name: "pause",
			call: func() error {
				_, err := commands.PauseInstruction(context.Background(), PauseInstructionInput{ID: "x70", Node: "x17"})
				return err
			},
			want: func(t *testing.T) {
				if store.pauseInstruction.Record.Kind != "pause" {
					t.Fatalf("kind = %q, want pause", store.pauseInstruction.Record.Kind)
				}
			},
		},
		{
			name: "release expired",
			call: func() error {
				_, err := commands.ReleaseExpiredLeaseInstruction(context.Background(), LeaseInstructionInput{
					ID:    "x71",
					Lease: "x16",
				})
				return err
			},
			want: func(t *testing.T) {
				if store.releaseExpiredInstruction.Record.Kind != "release_expired_lease" {
					t.Fatalf("kind = %q, want release_expired_lease", store.releaseExpiredInstruction.Record.Kind)
				}
			},
		},
		{
			name: "force release",
			call: func() error {
				_, err := commands.ForceReleaseLeaseInstruction(context.Background(), LeaseInstructionInput{
					ID:    "x72",
					Lease: "x16",
				})
				return err
			},
			want: func(t *testing.T) {
				if store.forceReleaseInstruction.Record.Kind != "force_release_lease" {
					t.Fatalf("kind = %q, want force_release_lease", store.forceReleaseInstruction.Record.Kind)
				}
			},
		},
		{
			name: "move item",
			call: func() error {
				_, err := commands.MoveItemInstruction(context.Background(), MoveItemInstructionInput{
					ID:       "x73",
					WorkItem: "x08",
					Source:   "x15",
					Target:   "x68",
				})
				return err
			},
			want: func(t *testing.T) {
				if store.moveItemInstruction.WorkItem != workitem.ID("x08") {
					t.Fatalf("work item = %q, want x08", store.moveItemInstruction.WorkItem)
				}
			},
		},
		{
			name: "move entries",
			call: func() error {
				_, err := commands.MoveEntriesInstruction(context.Background(), MoveEntriesInstructionInput{
					ID:      "x74",
					Entries: []channel.EntryID{"x31"},
				})
				return err
			},
			want: func(t *testing.T) {
				if store.moveEntriesInstruction.Entries[0] != channel.EntryID("x31") {
					t.Fatalf("entries = %v, want x31", store.moveEntriesInstruction.Entries)
				}
			},
		},
		{
			name: "move available",
			call: func() error {
				_, err := commands.MoveAvailableInstruction(context.Background(), MoveAvailableInstructionInput{
					ID:    "x75",
					Limit: 1,
				})
				return err
			},
			want: func(t *testing.T) {
				if store.moveAvailableInstruction.Limit != 1 {
					t.Fatalf("limit = %d, want 1", store.moveAvailableInstruction.Limit)
				}
			},
		},
		{
			name: "drop",
			call: func() error {
				_, err := commands.DropInstruction(context.Background(), ItemsInstructionInput{
					ID:        "x76",
					WorkItems: []workitem.ID{"x08"},
				})
				return err
			},
			want: func(t *testing.T) {
				if store.dropInstruction.Events[0] != "x86" {
					t.Fatalf("drop events = %v, want x86", store.dropInstruction.Events)
				}
			},
		},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			if err := call.call(); err != nil {
				t.Fatal(err)
			}
			call.want(t)
		})
	}
}

func TestFacadeReturnsChildBuildErrors(t *testing.T) {
	commands := NewCommands(&recordingStore{}, fixedClock{now: instant(0)}, &sequenceIDs{})

	if _, err := commands.PauseInstruction(context.Background(), PauseInstructionInput{Node: "x17"}); !errors.Is(err, ErrEmptyInstructionID) {
		t.Fatalf("pause instruction error = %v, want empty instruction id", err)
	}
	if _, err := commands.MoveAvailableInstruction(
		context.Background(),
		MoveAvailableInstructionInput{ID: "x70"},
	); !errors.Is(err, ErrInstructionLimit) {
		t.Fatalf("move available error = %v, want instruction limit", err)
	}
	if _, err := commands.Extend(context.Background(), ExtendInput{Token: " "}); !errors.Is(err, channel.ErrEmptyToken) {
		t.Fatalf("extend error = %v, want empty token", err)
	}
}

func mustLease(t *testing.T, id string) channel.Lease {
	t.Helper()

	lease, err := channel.NewLease(channel.LeaseInput{
		ID:        channel.LeaseID(id),
		Entry:     channel.EntryID("x32"),
		Channel:   registry.ChannelKey("x15"),
		WorkItem:  workitem.ID("x08"),
		GrantedAt: instant(0),
		ExpiresAt: instant(1),
	})
	if err != nil {
		t.Fatal(err)
	}
	return lease
}
