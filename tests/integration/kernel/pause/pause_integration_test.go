//go:build integration

package pause_test

import (
	"context"
	"errors"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/substrate"
	"testing"

	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/registry"
)

func TestPauseWritesExclusionAndNodeAuditOnly(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.InsertCapability(t, ctx, db, schema, "x18", "x12")
	before := fixture.RegistryFacts(t, ctx, db, schema)
	commands := fixture.CommandsFor(db, schema, "x45")

	result, err := commands.Pause(ctx, kernel.PauseInput{Node: registry.NodeKey("x17")})
	if err != nil {
		t.Fatal(err)
	}

	if !result.Paused {
		t.Fatal("pause result = false, want true")
	}
	after := fixture.RegistryFacts(t, ctx, db, schema)
	if after != before {
		t.Fatalf("registry facts changed\nbefore=%s\nafter=%s", before, after)
	}
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM routing_exclusions WHERE node_key = 'x17'`, 1)
	fixture.RequireScalar(t, ctx, db, schema, `
SELECT count(*)
FROM journal_events
WHERE coordinate_kind = 'node'
  AND coordinate_key = 'x17'
  AND event_kind_key = 'x45'`, 1)
}

func TestPauseFailsClosedWhenItWouldStrandNeedKind(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	commands := fixture.CommandsFor(db, schema, "x45")

	_, err := commands.Pause(ctx, kernel.PauseInput{Node: registry.NodeKey("x17")})

	if !errors.Is(err, kernel.ErrUnsafePause) {
		t.Fatalf("error = %v, want unsafe pause", err)
	}
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM routing_exclusions`, 0)
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM journal_events WHERE coordinate_kind = 'node'`, 0)
}
