package instructions

import (
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"time"
)

const (
	ExpiredLease   LeaseWindow = "lease_expired"
	UnexpiredLease LeaseWindow = "lease_unexpired"
)

type LeaseWindow string

type LeaseInput struct {
	ID    ID
	Lease channel.LeaseID
}

type LeaseCommand struct {
	Record Record
	Lease  channel.LeaseID
	Event  journal.EventID
}

type LeaseFacts struct {
	Lease channel.Lease
	Found bool
}

func ReleaseExpiredLease(input LeaseInput, clock Clock, ids IDs) (LeaseCommand, error) {
	return leaseCommand("release_expired_lease", input, clock, ids)
}

func ForceReleaseLease(input LeaseInput, clock Clock, ids IDs) (LeaseCommand, error) {
	return leaseCommand("force_release_lease", input, clock, ids)
}

func ApplyReleaseExpiredLease(command LeaseCommand, facts LeaseFacts) (Application, error) {
	return applyLease(command, facts, ExpiredLease)
}

func ApplyForceReleaseLease(command LeaseCommand, facts LeaseFacts) (Application, error) {
	return applyLease(command, facts, UnexpiredLease)
}

func leaseCommand(kind string, input LeaseInput, clock Clock, ids IDs) (LeaseCommand, error) {
	if err := validateID(input.ID); err != nil {
		return LeaseCommand{}, err
	}
	return LeaseCommand{
		Record: record(kind, input, clock),
		Lease:  input.Lease,
		Event:  eventID(ids),
	}, nil
}

func applyLease(command LeaseCommand, facts LeaseFacts, window LeaseWindow) (Application, error) {
	scope := newScope(command.Record, command.Event, leaseAudit(command.Lease, window))
	coordinate := journal.LeaseCoordinate(command.Lease.String())
	if !facts.Found {
		return rejected(scope, coordinate, "lease_exists")
	}
	if !leaseMatchesWindow(facts.Lease, command.Record.RecordedAt, window) {
		return rejected(scope, coordinate, string(window))
	}
	return applied(scope, coordinate, []string{command.Lease.String()}, Effects{
		Releases: []channel.LeaseID{command.Lease},
	})
}

func leaseAudit(lease channel.LeaseID, window LeaseWindow) audit {
	return newAudit(
		map[string]any{"lease_id": lease.String()},
		"lease_exists",
		string(window),
	)
}

func leaseMatchesWindow(lease channel.Lease, now time.Time, window LeaseWindow) bool {
	switch window {
	case ExpiredLease:
		return !lease.ExpiresAt().After(now)
	case UnexpiredLease:
		return lease.ExpiresAt().After(now)
	default:
		return false
	}
}
