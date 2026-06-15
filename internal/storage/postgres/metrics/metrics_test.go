package metrics

import (
	"context"
	"testing"
	"time"
)

func TestAggregatesScansAvailableLeaseCounts(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	store := Store{
		query: func(ctx context.Context, query string, args ...any) rowScanner {
			requireQueryTime(t, args, now)
			return aggregateRow{available: 3, held: 2, expired: 1}
		},
	}

	got, err := store.Aggregates(context.Background(), now)

	if err != nil {
		t.Fatal(err)
	}
	if got.AvailableDepth != 3 {
		t.Fatalf("available depth = %d, want 3", got.AvailableDepth)
	}
	if got.LeasesHeld != 2 {
		t.Fatalf("leases held = %d, want 2", got.LeasesHeld)
	}
	if got.LeasesExpired != 1 {
		t.Fatalf("leases expired = %d, want 1", got.LeasesExpired)
	}
}

func requireQueryTime(t *testing.T, args []any, want time.Time) {
	t.Helper()

	if len(args) != 1 {
		t.Fatalf("query arg count = %d, want 1", len(args))
	}
	got, ok := args[0].(time.Time)
	if !ok || !got.Equal(want) {
		t.Fatalf("query time = %v, %v; want %v, true", args[0], ok, want)
	}
}

type aggregateRow struct {
	available int
	held      int
	expired   int
}

func (row aggregateRow) Scan(targets ...any) error {
	*targets[0].(*int) = row.available
	*targets[1].(*int) = row.held
	*targets[2].(*int) = row.expired
	return nil
}
