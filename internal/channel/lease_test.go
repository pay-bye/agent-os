package channel

import (
	"errors"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestNewLeaseRejectsInvalidInput(t *testing.T) {
	for _, test := range invalidLeaseCases() {
		assertInvalidLease(t, test)
	}
}

func TestNewLeaseReportsFields(t *testing.T) {
	lease, err := NewLease(validLeaseInput())
	if err != nil {
		t.Fatal(err)
	}

	if lease.ID() != LeaseID("x16") {
		t.Fatalf("lease identity = %q, want x16", lease.ID())
	}
	if lease.Entry() != EntryID("x32") {
		t.Fatalf("entry identity = %q, want x32", lease.Entry())
	}
	if lease.WorkItem() != workitem.ID("x08") {
		t.Fatalf("work item = %q, want x08", lease.WorkItem())
	}
}

func validLeaseInput(changes ...func(*LeaseInput)) LeaseInput {
	input := LeaseInput{
		ID:        LeaseID("x16"),
		Entry:     EntryID("x32"),
		Channel:   registry.ChannelKey("x15"),
		WorkItem:  workitem.ID("x08"),
		GrantedAt: instant(0),
		ExpiresAt: instant(1),
	}
	for _, change := range changes {
		change(&input)
	}
	return input
}

func invalidLeaseCases() []invalidLeaseCase {
	return []invalidLeaseCase{
		{name: "empty identity", input: leaseWithoutID(), want: ErrEmptyLeaseID},
		{name: "empty entry", input: leaseWithoutEntry(), want: ErrEmptyEntryID},
		{name: "empty channel", input: leaseWithoutChannel(), want: ErrEmptyChannelKey},
		{name: "empty work item", input: leaseWithoutWorkItem(), want: ErrEmptyWorkItemID},
		{name: "zero granted time", input: leaseWithoutGrantTime(), want: ErrMissingGrantedAt},
		{name: "zero expiry time", input: leaseWithoutExpiryTime(), want: ErrMissingExpiresAt},
		{name: "expiry before grant", input: leaseExpiringAtGrant(), want: ErrInvalidLeaseWindow},
	}
}

func leaseWithoutID() LeaseInput {
	return validLeaseInput(func(input *LeaseInput) { input.ID = "" })
}

func leaseWithoutEntry() LeaseInput {
	return validLeaseInput(func(input *LeaseInput) { input.Entry = "" })
}

func leaseWithoutChannel() LeaseInput {
	return validLeaseInput(func(input *LeaseInput) { input.Channel = "" })
}

func leaseWithoutWorkItem() LeaseInput {
	return validLeaseInput(func(input *LeaseInput) { input.WorkItem = "" })
}

func leaseWithoutGrantTime() LeaseInput {
	return validLeaseInput(func(input *LeaseInput) { input.GrantedAt = time.Time{} })
}

func leaseWithoutExpiryTime() LeaseInput {
	return validLeaseInput(func(input *LeaseInput) { input.ExpiresAt = time.Time{} })
}

func leaseExpiringAtGrant() LeaseInput {
	return validLeaseInput(func(input *LeaseInput) { input.ExpiresAt = input.GrantedAt })
}

type invalidLeaseCase struct {
	name  string
	input LeaseInput
	want  error
}

func assertInvalidLease(t *testing.T, test invalidLeaseCase) {
	t.Helper()

	t.Run(test.name, func(t *testing.T) {
		_, err := NewLease(test.input)

		if !errors.Is(err, test.want) {
			t.Fatalf("error = %v, want %v", err, test.want)
		}
	})
}
