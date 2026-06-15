//go:build integration

package resolution_test

import (
	"context"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/substrate"
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
)

func TestAckRollsBackWhenNextRouteAppendFails(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.SubmitTwoNeeds(t, ctx, db, schema)
	token := fixture.ClaimAlpha(t, ctx, db, schema)
	commands := fixture.CommandsFor(db, schema, "x31", "x26", "x33")

	_, err := commands.Ack(ctx, kernel.ResolutionInput{Lease: channel.LeaseID("x16"), Token: token})

	if err == nil {
		t.Fatal("expected duplicate route event to fail")
	}
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM leases WHERE id = 'x16'`, int64(1))
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM journal_events WHERE event_kind_key = 'x43'`, int64(0))
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM channel_entries WHERE channel_key = 'x68'`, int64(0))
}
