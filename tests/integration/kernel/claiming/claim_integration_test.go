//go:build integration

package claiming_test

import (
	"context"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/substrate"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestClaimReturnsLeaseWithPayload(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.SubmitRoutedItem(t, ctx, db, schema)
	commands := fixture.CommandsFor(db, schema)

	result, err := commands.Claim(ctx, kernel.ClaimInput{
		Channel:       registry.ChannelKey("x15"),
		Lease:         channel.LeaseID("x16"),
		LeaseDuration: 10 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Empty {
		t.Fatal("claim returned empty queue")
	}
	if result.WorkItem != workitem.ID("x08") {
		t.Fatalf("work item = %q, want x08", result.WorkItem)
	}
	if string(result.Payload) != `{"value": "x75"}` && string(result.Payload) != `{"value":"x75"}` {
		t.Fatalf("payload = %s, want submitted item payload", result.Payload)
	}
}

func TestClaimReturnsEmptyResultForEmptyChannel(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	commands := fixture.CommandsFor(db, schema)

	result, err := commands.Claim(ctx, kernel.ClaimInput{
		Channel:       registry.ChannelKey("x15"),
		Lease:         channel.LeaseID("x16"),
		LeaseDuration: 10 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !result.Empty {
		t.Fatalf("claim result = %+v, want empty", result)
	}
}
