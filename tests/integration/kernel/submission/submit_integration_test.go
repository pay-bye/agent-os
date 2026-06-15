//go:build integration

package submission_test

import (
	"context"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/substrate"
	"testing"

	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestSubmitPersistsHistoryAndRoutesFirstNeed(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	commands := fixture.CommandsFor(db, schema, "x25", "x27", "x30", "x32")

	result, err := commands.Submit(ctx, kernel.SubmitInput{
		ID:      workitem.ID("x08"),
		Kind:    registry.ItemKindKey("x08"),
		Payload: []byte(`{"value":"x75"}`),
		DeclaredNeeds: []workitem.DeclaredNeedInput{
			{Kind: registry.NeedKindKey("x12"), Payload: []byte(`{"value":"x76"}`)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !result.Routed || result.Channel != registry.ChannelKey("x15") {
		t.Fatalf("route result = %+v, want x15", result)
	}
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM work_items`, int64(1))
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM channel_entries WHERE channel_key = 'x15'`, int64(1))
	fixture.RequireEventKinds(t, ctx, db, schema, "x40", "x41", "x42")
}

func TestSubmitWithoutNeedsDoesNotCreateChannelEntry(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	commands := fixture.CommandsFor(db, schema, "x25", "x30", "x32")

	result, err := commands.Submit(ctx, kernel.SubmitInput{
		ID:      workitem.ID("x08"),
		Kind:    registry.ItemKindKey("x08"),
		Payload: []byte(`{"value":"x75"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Routed {
		t.Fatalf("route result = %+v, want no route", result)
	}
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM channel_entries`, int64(0))
}

func TestSubmitRollsBackWhenChannelEnqueueFails(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.InsertConflictingEntry(t, ctx, db, schema, "x32")
	commands := fixture.CommandsFor(db, schema, "x25", "x27", "x30", "x32")

	_, err := commands.Submit(ctx, fixture.SubmissionInput())

	if err == nil {
		t.Fatal("expected duplicate channel entry identity to fail submit")
	}
	fixture.RequireNoSubmittedWork(t, ctx, db, schema)
}
