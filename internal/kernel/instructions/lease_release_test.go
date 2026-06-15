package instructions

import (
	"slices"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/journal/payloads"
)

func TestLeaseWindowSeparatesExpiredAndHeld(t *testing.T) {
	expired := leaseWithExpiry(t, instant(1))
	held := leaseWithExpiry(t, instant(3))

	if !leaseMatchesWindow(expired, instant(2), ExpiredLease) {
		t.Fatal("expired lease did not match expired window")
	}
	if leaseMatchesWindow(held, instant(2), ExpiredLease) {
		t.Fatal("held lease matched expired window")
	}
	if !leaseMatchesWindow(held, instant(2), UnexpiredLease) {
		t.Fatal("held lease did not match unexpired window")
	}
	if leaseMatchesWindow(expired, instant(2), UnexpiredLease) {
		t.Fatal("expired lease matched unexpired window")
	}
}

func TestReleaseExpiredLeaseCommandBuildsStoreCommand(t *testing.T) {
	command, err := ReleaseExpiredLease(LeaseInput{
		ID:    "x70",
		Lease: "x16",
	}, fixedClock{now: instant(0)}, &sequenceIDs{values: []string{"x80"}})
	if err != nil {
		t.Fatal(err)
	}

	if command.Record.Kind != "release_expired_lease" {
		t.Fatalf("kind = %q, want release_expired_lease", command.Record.Kind)
	}
	if command.Lease != channel.LeaseID("x16") {
		t.Fatalf("lease = %q, want x16", command.Lease)
	}
}

func TestForceReleaseLeaseCommandBuildsStoreCommand(t *testing.T) {
	command, err := ForceReleaseLease(LeaseInput{
		ID:    "x70",
		Lease: "x16",
	}, fixedClock{now: instant(0)}, &sequenceIDs{values: []string{"x80"}})
	if err != nil {
		t.Fatal(err)
	}

	if command.Record.Kind != "force_release_lease" {
		t.Fatalf("kind = %q, want force_release_lease", command.Record.Kind)
	}
	if command.Event != journal.EventID("x80") {
		t.Fatalf("event = %q, want x80", command.Event)
	}
}

func TestApplyReleaseExpiredLeaseBuildsReleasePlan(t *testing.T) {
	command := LeaseCommand{Record: recordAt("x70"), Lease: "x16", Event: "x80"}
	facts := LeaseFacts{Lease: leaseWithExpiry(t, instant(1)), Found: true}

	application, err := ApplyReleaseExpiredLease(command, facts)

	if err != nil {
		t.Fatal(err)
	}
	requireResult(t, application.Result, Applied, "")
	requireOutcome(t, application.Outcomes[0], payloads.InstructionAppliedKind)
	if !slices.Equal(application.Effects.Releases, []channel.LeaseID{"x16"}) {
		t.Fatalf("releases = %v, want x16", application.Effects.Releases)
	}
}

func TestApplyForceReleaseLeaseRejectsExpiredLease(t *testing.T) {
	command := LeaseCommand{Record: recordAt("x70"), Lease: "x16", Event: "x80"}
	facts := LeaseFacts{Lease: leaseWithExpiry(t, instant(1)), Found: true}

	application, err := ApplyForceReleaseLease(command, facts)

	if err != nil {
		t.Fatal(err)
	}
	requireResult(t, application.Result, PreconditionFailed, "lease_unexpired")
	requireOutcome(t, application.Outcomes[0], payloads.InstructionRejectedKind)
}

func leaseWithExpiry(t *testing.T, expiresAt time.Time) channel.Lease {
	t.Helper()

	lease, err := channel.NewLease(channel.LeaseInput{
		ID:        "x16",
		Entry:     "x31",
		Channel:   "x15",
		WorkItem:  "x08",
		GrantedAt: instant(0),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	return lease
}
