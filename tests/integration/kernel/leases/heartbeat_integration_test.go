//go:build integration

package leases_test

import (
	"context"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/substrate"
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
)

func TestHeartbeatReadsLeaseWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.SubmitRoutedItem(t, ctx, db, schema)
	token := fixture.ClaimAlpha(t, ctx, db, schema)
	before := fixture.ReadFacts(t, ctx, db, schema)
	commands := fixture.CommandsFor(db, schema)

	result, err := commands.Heartbeat(ctx, kernel.HeartbeatInput{Lease: channel.LeaseID("x16"), Token: token})
	after := fixture.ReadFacts(t, ctx, db, schema)

	if err != nil {
		t.Fatal(err)
	}
	if result.Lease.ID() != channel.LeaseID("x16") {
		t.Fatalf("heartbeat lease = %q, want x16", result.Lease.ID())
	}
	fixture.RequireUnchangedFacts(t, before, after)
}
