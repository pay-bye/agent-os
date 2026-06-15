//go:build integration

package kernel_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/substrate"
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
)

func TestLeaseCommandsRejectWrongTokenWithoutMutation(t *testing.T) {
	cases := []wrongTokenCase{
		{name: "ack", seed: seedRoutedLeaseWithNextNeed, command: ackWithWrongToken},
		{name: "nack", seed: seedRoutedLease, command: nackWithWrongToken},
		{name: "extend", seed: seedRoutedLease, command: extendWithWrongToken},
		{name: "heartbeat", seed: seedRoutedLease, command: heartbeatWithWrongToken},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, schema := fixture.MigratedSchema(t, ctx)
			tc.seed(t, ctx, db, schema)
			before := fixture.ReadFacts(t, ctx, db, schema)
			commands := fixture.CommandsFor(db, schema, "x31", "x29", "x33")

			err := tc.command(ctx, commands)
			after := fixture.ReadFacts(t, ctx, db, schema)

			if !errors.Is(err, channel.ErrInvalidLease) {
				t.Fatalf("error = %v, want invalid lease", err)
			}
			fixture.RequireUnchangedFacts(t, before, after)
		})
	}
}

type wrongTokenCase struct {
	name    string
	seed    func(*testing.T, context.Context, *sql.DB, string)
	command func(context.Context, kernel.Commands) error
}

func seedRoutedLease(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
	t.Helper()

	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.SubmitRoutedItem(t, ctx, db, schema)
	fixture.ClaimAlpha(t, ctx, db, schema)
}

func seedRoutedLeaseWithNextNeed(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
	t.Helper()

	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.SubmitTwoNeeds(t, ctx, db, schema)
	fixture.ClaimAlpha(t, ctx, db, schema)
}

func ackWithWrongToken(ctx context.Context, commands kernel.Commands) error {
	_, err := commands.Ack(ctx, kernel.ResolutionInput{Lease: channel.LeaseID("x16"), Token: wrongToken()})
	return err
}

func nackWithWrongToken(ctx context.Context, commands kernel.Commands) error {
	_, err := commands.Nack(ctx, kernel.ResolutionInput{Lease: channel.LeaseID("x16"), Token: wrongToken()})
	return err
}

func extendWithWrongToken(ctx context.Context, commands kernel.Commands) error {
	_, err := commands.Extend(ctx, kernel.ExtendInput{
		Lease:     channel.LeaseID("x16"),
		Token:     wrongToken(),
		ExpiresAt: fixture.Instant(20),
	})
	return err
}

func heartbeatWithWrongToken(ctx context.Context, commands kernel.Commands) error {
	_, err := commands.Heartbeat(ctx, kernel.HeartbeatInput{Lease: channel.LeaseID("x16"), Token: wrongToken()})
	return err
}

func wrongToken() channel.Token {
	return channel.Token("x-wrong-token")
}
