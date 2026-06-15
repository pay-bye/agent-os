//go:build integration

package submission_test

import (
	"context"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/substrate"
	"testing"
)

func TestSubmitRollsBackWhenJournalAppendFails(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.InsertConflictingEvent(t, ctx, db, schema, "x25")
	commands := fixture.CommandsFor(db, schema, "x25", "x27", "x30", "x32")

	_, err := commands.Submit(ctx, fixture.SubmissionInput())

	if err == nil {
		t.Fatal("expected duplicate event identity to fail submit")
	}
	fixture.RequireNoSubmittedWork(t, ctx, db, schema)
}
