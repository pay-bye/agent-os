//go:build integration

package leases_test

import (
	"context"
	"errors"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/substrate"
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
)

func TestExtendAndHeartbeatUseActiveLease(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.SubmitRoutedItem(t, ctx, db, schema)
	token := fixture.ClaimAlpha(t, ctx, db, schema)
	commands := fixture.CommandsFor(db, schema)

	extended, err := commands.Extend(ctx, kernel.ExtendInput{
		Lease:     channel.LeaseID("x16"),
		Token:     token,
		ExpiresAt: fixture.Instant(20),
	})
	if err != nil {
		t.Fatal(err)
	}
	heartbeat, err := commands.Heartbeat(ctx, kernel.HeartbeatInput{Lease: channel.LeaseID("x16"), Token: token})
	if err != nil {
		t.Fatal(err)
	}

	if !extended.Lease.ExpiresAt().Equal(fixture.Instant(20)) {
		t.Fatalf("extended expiry = %s, want %s", extended.Lease.ExpiresAt(), fixture.Instant(20))
	}
	if heartbeat.Lease.ID() != channel.LeaseID("x16") {
		t.Fatalf("heartbeat lease = %q, want x16", heartbeat.Lease.ID())
	}
}

func TestExtendRejectsInactiveOrEarlierLeaseThroughKernel(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.SubmitRoutedItem(t, ctx, db, schema)
	token := fixture.ClaimAlpha(t, ctx, db, schema)
	commands := fixture.CommandsFor(db, schema)
	expiredCommands := fixture.CommandsAt(db, schema, fixture.Instant(20))

	_, missingErr := commands.Extend(ctx, kernel.ExtendInput{
		Lease:     channel.LeaseID("x58"),
		Token:     token,
		ExpiresAt: fixture.Instant(20),
	})
	_, staleErr := commands.Extend(ctx, kernel.ExtendInput{
		Lease:     channel.LeaseID("x16"),
		Token:     token,
		ExpiresAt: fixture.Instant(5),
	})
	_, expiredErr := expiredCommands.Extend(ctx, kernel.ExtendInput{
		Lease:     channel.LeaseID("x16"),
		Token:     token,
		ExpiresAt: fixture.Instant(30),
	})

	if !errors.Is(missingErr, channel.ErrInvalidLease) {
		t.Fatalf("missing lease error = %v, want invalid lease", missingErr)
	}
	if !errors.Is(staleErr, channel.ErrNonIncreasingExtension) {
		t.Fatalf("stale extension error = %v, want non-increasing extension", staleErr)
	}
	if !errors.Is(expiredErr, channel.ErrExpiredLease) {
		t.Fatalf("expired extension error = %v, want expired lease", expiredErr)
	}
	fixture.RequireLeaseExpiry(t, ctx, db, schema, "x16", fixture.Instant(10))
}

func TestAckRejectsMissingLease(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	commands := fixture.CommandsFor(db, schema, "x31", "x30", "x34")

	_, err := commands.Ack(ctx, kernel.ResolutionInput{
		Lease: channel.LeaseID("x58"),
		Token: channel.Token("x-token"),
	})

	if !errors.Is(err, channel.ErrInvalidLease) {
		t.Fatalf("error = %v, want invalid lease", err)
	}
}
