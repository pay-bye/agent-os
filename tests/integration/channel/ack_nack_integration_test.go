//go:build integration

package channel_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/channel"
	channelstore "github.com/pay-bye/agent-os/internal/storage/postgres/channel"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

func TestPrepareAckAndNackRequireUnexpiredLease(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	insertVocabulary(t, ctx, tx)
	store := channelstore.New(tx)
	now := instant(0)
	enqueue(t, ctx, store, "x32", "x08", now)
	lease := dequeue(t, ctx, store, "x16", now.Add(time.Minute))

	ack, err := store.PrepareAck(ctx, lease.ID(), now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	nack, err := store.PrepareNack(ctx, lease.ID(), now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	if ack.Kind() != channel.Ack {
		t.Fatalf("ack kind = %q, want ack", ack.Kind())
	}
	if nack.Kind() != channel.Nack {
		t.Fatalf("nack kind = %q, want nack", nack.Kind())
	}
}

func TestPrepareRejectsMissingAndExpiredLease(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	insertVocabulary(t, ctx, tx)
	store := channelstore.New(tx)
	now := instant(0)
	enqueue(t, ctx, store, "x32", "x08", now)
	lease := dequeue(t, ctx, store, "x16", now.Add(time.Minute))

	_, missingErr := store.PrepareAck(ctx, channel.LeaseID("x58"), now)
	_, expiredErr := store.PrepareNack(ctx, lease.ID(), now.Add(20*time.Minute))

	if !errors.Is(missingErr, channel.ErrInvalidLease) {
		t.Fatalf("missing lease error = %v, want invalid lease", missingErr)
	}
	if !errors.Is(expiredErr, channel.ErrExpiredLease) {
		t.Fatalf("expired lease error = %v, want expired lease", expiredErr)
	}
}
