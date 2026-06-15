//go:build integration

package routing_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/substrate"
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestSubmitRollsBackWhenRoutingHasNoRule(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	removeRoute(t, ctx, db, schema, "x12")
	commands := fixture.CommandsFor(db, schema, "x25", "x27", "x30", "x32")

	_, err := commands.Submit(ctx, fixture.SubmissionInput())

	if !errors.Is(err, registry.ErrNoRoute) {
		t.Fatalf("error = %v, want no route", err)
	}
	fixture.RequireNoSubmittedWork(t, ctx, db, schema)
}

func TestSubmitAddressedNeedRoutesToTargetOutsideDefaultOrder(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.InsertCapability(t, ctx, db, schema, "x18", "x12")
	commands := fixture.CommandsFor(db, schema, "x25", "x27", "x30", "x32")

	result, err := commands.Submit(ctx, kernel.SubmitInput{
		ID:      workitem.ID("x08"),
		Kind:    registry.ItemKindKey("x08"),
		Payload: []byte(`{"value":"x75"}`),
		DeclaredNeeds: []workitem.DeclaredNeedInput{{
			Kind:    registry.NeedKindKey("x12"),
			Target:  registry.NodeKey("x18"),
			Payload: []byte(`{"target_node":"x17","value":"x76"}`),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !result.Routed || result.Channel != registry.ChannelKey("x68") {
		t.Fatalf("route result = %+v, want x68", result)
	}
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM channel_entries WHERE channel_key = 'x68'`, int64(1))
}

func TestSubmitAddressedNeedFailsClosedWhenTargetIsAbsent(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	commands := fixture.CommandsFor(db, schema, "x25", "x27", "x30", "x32")

	_, err := commands.Submit(ctx, targetedSubmission("x99"))

	if !errors.Is(err, kernel.ErrRouteTargetAbsent) {
		t.Fatalf("error = %v, want absent route target", err)
	}
	fixture.RequireNoSubmittedWork(t, ctx, db, schema)
}

func TestSubmitAddressedNeedFailsClosedWhenTargetIsIncapable(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	commands := fixture.CommandsFor(db, schema, "x25", "x27", "x30", "x32")

	_, err := commands.Submit(ctx, targetedSubmission("x18"))

	if !errors.Is(err, kernel.ErrRouteTargetIncapable) {
		t.Fatalf("error = %v, want incapable route target", err)
	}
	fixture.RequireNoSubmittedWork(t, ctx, db, schema)
}

func TestSubmitAddressedNeedFailsClosedWhenTargetIsExcluded(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	insertExclusion(t, ctx, db, schema, "x17")
	commands := fixture.CommandsFor(db, schema, "x25", "x27", "x30", "x32")

	_, err := commands.Submit(ctx, targetedSubmission("x17"))

	if !errors.Is(err, kernel.ErrRouteTargetExcluded) {
		t.Fatalf("error = %v, want excluded route target", err)
	}
	fixture.RequireNoSubmittedWork(t, ctx, db, schema)
}

func TestSubmitDefaultNeedSkipsExcludedRoute(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.InsertCapability(t, ctx, db, schema, "x18", "x12")
	insertRoute(t, ctx, db, schema, "x12", "x18", 2)
	insertExclusion(t, ctx, db, schema, "x17")
	commands := fixture.CommandsFor(db, schema, "x25", "x27", "x30", "x32")

	result, err := commands.Submit(ctx, fixture.SubmissionInput())
	if err != nil {
		t.Fatal(err)
	}

	if !result.Routed || result.Channel != registry.ChannelKey("x68") {
		t.Fatalf("route result = %+v, want x68", result)
	}
}

func TestSubmitDefaultNeedFailsClosedWhenAllRoutesAreExcluded(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	insertExclusion(t, ctx, db, schema, "x17")
	commands := fixture.CommandsFor(db, schema, "x25", "x27", "x30", "x32")

	_, err := commands.Submit(ctx, fixture.SubmissionInput())

	if !errors.Is(err, registry.ErrNoRoute) {
		t.Fatalf("error = %v, want no route", err)
	}
	fixture.RequireNoSubmittedWork(t, ctx, db, schema)
}

func TestAckCleansLeaseAndRoutesNextOutstandingNeed(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.SubmitTwoNeeds(t, ctx, db, schema)
	token := fixture.ClaimAlpha(t, ctx, db, schema)
	commands := fixture.CommandsFor(db, schema, "x31", "x29", "x33")

	result, err := commands.Ack(ctx, kernel.ResolutionInput{Lease: channel.LeaseID("x16"), Token: token})
	if err != nil {
		t.Fatal(err)
	}

	if !result.Resolved || !result.Routed || result.Channel != registry.ChannelKey("x68") {
		t.Fatalf("ack result = %+v, want routed x68", result)
	}
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM leases`, int64(0))
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM channel_entries WHERE channel_key = 'x15'`, int64(0))
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM channel_entries WHERE channel_key = 'x68'`, int64(1))
	fixture.RequireEventKinds(t, ctx, db, schema, "x40", "x41", "x41", "x42", "x43", "x42")
}

func TestNackRetriesOnlyWhenNeedIsDeclared(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.SubmitRoutedItem(t, ctx, db, schema)
	token := fixture.ClaimAlpha(t, ctx, db, schema)
	commands := fixture.CommandsFor(db, schema, "x53")

	result, err := commands.Nack(ctx, kernel.ResolutionInput{Lease: channel.LeaseID("x16"), Token: token})
	if err != nil {
		t.Fatal(err)
	}

	if !result.Resolved || result.Routed {
		t.Fatalf("nack result = %+v, want resolved without implicit retry", result)
	}
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM channel_entries`, int64(0))
}

func targetedSubmission(target string) kernel.SubmitInput {
	input := fixture.SubmissionInput()
	input.DeclaredNeeds[0].Target = registry.NodeKey(target)
	return input
}

func removeRoute(t *testing.T, ctx context.Context, db *sql.DB, schema string, need string) {
	t.Helper()

	_, err := db.ExecContext(ctx, `
DELETE FROM `+schema+`.routing_rules
WHERE need_kind_key = $1`, need)
	if err != nil {
		t.Fatal(err)
	}
}

func insertRoute(
	t *testing.T,
	ctx context.Context,
	db *sql.DB,
	schema string,
	need string,
	node string,
	order int,
) {
	t.Helper()

	_, err := db.ExecContext(ctx, `
INSERT INTO `+schema+`.routing_rules (need_kind_key, node_key, rule_order)
VALUES ($1, $2, $3)`, need, node, order)
	if err != nil {
		t.Fatal(err)
	}
}

func insertExclusion(t *testing.T, ctx context.Context, db *sql.DB, schema string, node string) {
	t.Helper()

	_, err := db.ExecContext(ctx, `
INSERT INTO `+schema+`.routing_exclusions (node_key) VALUES ($1)`, node)
	if err != nil {
		t.Fatal(err)
	}
}
