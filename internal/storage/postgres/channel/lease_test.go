package channel

import (
	"context"
	"errors"
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestLeaseFromRowMapsLease(t *testing.T) {
	lease, err := LeaseFromRow(rowValues{values: []any{
		"x16",
		"x32",
		"x08",
		"x15",
		instant(0),
		instant(1),
	}})
	if err != nil {
		t.Fatal(err)
	}

	if lease.ID() != channel.LeaseID("x16") {
		t.Fatalf("lease identity = %q, want x16", lease.ID())
	}
}

func TestLeaseFromRowReturnsEmptyQueue(t *testing.T) {
	_, err := LeaseFromRow(missingRow{})

	if !errors.Is(err, channel.ErrEmpty) {
		t.Fatalf("error = %v, want empty queue", err)
	}
}

func TestDequeueMapsGrantedLease(t *testing.T) {
	var gotArgs []any
	store := &Store{query: func(_ context.Context, _ string, args ...any) rowScanner {
		gotArgs = append([]any(nil), args...)
		return rowValues{values: []any{
			"x16",
			"x32",
			"x08",
			"x15",
			instant(0),
			instant(1),
		}}
	}}

	lease, err := store.Dequeue(context.Background(), registry.ChannelKey("x15"), channel.LeaseRequest{
		ID:          channel.LeaseID("x16"),
		TokenDigest: channel.Digest("GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14"),
		GrantedAt:   instant(0),
		ExpiresAt:   instant(1),
	})
	if err != nil {
		t.Fatal(err)
	}

	if lease.WorkItem() != workitem.ID("x08") {
		t.Fatalf("work item = %q, want x08", lease.WorkItem())
	}
	requireArgs(t, gotArgs,
		"x15",
		instant(0),
		"x16",
		instant(1),
		"GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14",
	)
}

func TestHeartbeatRejectsWrongDigest(t *testing.T) {
	store := &Store{query: func(context.Context, string, ...any) rowScanner {
		return rowValues{values: []any{
			"x16",
			"x32",
			"x08",
			"x15",
			instant(0),
			instant(1),
			"GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14",
		}}
	}}

	_, err := store.Heartbeat(
		context.Background(),
		channel.LeaseID("x16"),
		channel.Digest("wrong"),
		instant(0),
	)

	if !errors.Is(err, channel.ErrInvalidLease) {
		t.Fatalf("error = %v, want invalid lease", err)
	}
}
