package channel

import (
	"context"
	"errors"
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
)

func TestPrepareAckRejectsExpiredLease(t *testing.T) {
	store := &Store{query: func(context.Context, string, ...any) rowScanner {
		return rowValues{values: []any{instant(1)}}
	}}

	_, err := store.PrepareAck(context.Background(), channel.LeaseID("x16"), instant(2))

	if !errors.Is(err, channel.ErrExpiredLease) {
		t.Fatalf("error = %v, want expired lease", err)
	}
}

func TestPrepareNackReturnsPreparation(t *testing.T) {
	store := &Store{query: func(context.Context, string, ...any) rowScanner {
		return rowValues{values: []any{instant(2)}}
	}}

	item, err := store.PrepareNack(context.Background(), channel.LeaseID("x16"), instant(1))
	if err != nil {
		t.Fatal(err)
	}

	if item.Kind() != channel.Nack {
		t.Fatalf("kind = %q, want nack", item.Kind())
	}
}
