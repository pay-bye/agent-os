package postgres

import (
	"context"
	"testing"
	"time"
)

func TestOperationsPressureScansQueueAndLeaseCounts(t *testing.T) {
	var gotQuery string
	var gotArgs []any
	operations := &Operations{query: func(_ context.Context, query string, args ...any) rowScanner {
		gotQuery = query
		gotArgs = append([]any(nil), args...)
		return rowValues{values: []any{12, 7, 3, 1, 84}}
	}}
	now := instant(0)

	got, err := operations.Pressure(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}

	requireQuery(t, gotQuery, pressureQuery)
	requireArgs(t, gotArgs, now)
	if got.Depth != 12 || got.Available != 7 || got.Held != 3 || got.Expired != 1 {
		t.Fatalf("pressure = %+v", got)
	}
	if got.OldestAvailableAgeSeconds != 84 {
		t.Fatalf("oldest age = %d, want 84", got.OldestAvailableAgeSeconds)
	}
}

func TestOperationsJournalUsesRequestedWindow(t *testing.T) {
	var gotArgs []any
	operations := &Operations{query: func(_ context.Context, _ string, args ...any) rowScanner {
		gotArgs = append([]any(nil), args...)
		return rowValues{values: []any{120}}
	}}
	now := instant(5)

	got, err := operations.Journal(context.Background(), now, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	requireArgs(t, gotArgs, now.Add(-5*time.Minute), now)
	if got.Appends != 120 || got.WindowSeconds != 300 {
		t.Fatalf("journal window = %+v", got)
	}
}

func TestOperationsChannelsScansLocatorRows(t *testing.T) {
	var gotArgs []any
	operations := &Operations{rows: func(_ context.Context, _ string, args ...any) (rowsScanner, error) {
		gotArgs = append([]any(nil), args...)
		return &rowsValues{rows: [][]any{
			{"x15", "x17", 4, 2, 90},
			{"x16", "x18", 1, 0, 0},
		}}, nil
	}}
	now := instant(7)

	got, err := operations.Channels(context.Background(), now, ChannelQuery{
		Limit:            50,
		After:            "x14",
		OlderThanSeconds: 60,
	})
	if err != nil {
		t.Fatal(err)
	}

	requireArgs(t, gotArgs, now, "x14", 60, 50)
	requireChannels(t, got, []ChannelSummary{
		{Key: "x15", Node: "x17", Depth: 4, Available: 2, OldestAvailableAgeSeconds: 90},
		{Key: "x16", Node: "x18", Depth: 1, Available: 0, OldestAvailableAgeSeconds: 0},
	})
}

func TestOperationsChannelItemsScansLeaseState(t *testing.T) {
	var gotArgs []any
	operations := &Operations{rows: func(_ context.Context, _ string, args ...any) (rowsScanner, error) {
		gotArgs = append([]any(nil), args...)
		return &rowsValues{rows: [][]any{
			{"x31", "x08", "x15", "x17", instant(0), instant(1), 360, "x16", instant(2), instant(3)},
			{"x32", "x09", "x15", "x17", instant(4), instant(5), 120, nil, nil, nil},
		}}, nil
	}}
	now := instant(8)

	got, err := operations.ChannelItems(context.Background(), now, ChannelItemQuery{
		Channel:          "x15",
		Limit:            10,
		OlderThanSeconds: 30,
		Lease:            LeaseViewHeld,
	})
	if err != nil {
		t.Fatal(err)
	}

	requireArgs(t, gotArgs, now, "x15", 30, LeaseViewHeld, 10)
	requireChannelItems(t, got, []ChannelItem{
		{
			Entry: "x31", WorkItem: "x08", Channel: "x15", Node: "x17",
			EnqueuedAt: instant(0), AvailableAt: instant(1), AgeSeconds: 360,
			Lease: &Lease{ID: "x16", GrantedAt: instant(2), ExpiresAt: instant(3)},
		},
		{
			Entry: "x32", WorkItem: "x09", Channel: "x15", Node: "x17",
			EnqueuedAt: instant(4), AvailableAt: instant(5), AgeSeconds: 120,
		},
	})
}

func TestOperationsItemCombinesEntryLeaseAndNeed(t *testing.T) {
	var calls int
	operations := &Operations{query: func(context.Context, string, ...any) rowScanner {
		calls++
		if calls == 1 {
			return rowValues{values: []any{
				"x08", "x91", instant(0),
				"x31", "x08", "x15", "x17", instant(1), instant(2), int64(240),
				"x16", "x15", instant(3), instant(4),
			}}
		}
		return rowValues{values: []any{"x77", "x12", "x17", instant(5)}}
	}}

	got, err := operations.Item(context.Background(), instant(6), "x08")
	if err != nil {
		t.Fatal(err)
	}

	requireItem(t, got, ItemDetail{
		WorkItem: "x08",
		Kind:     "x91",
		Entry: &ItemEntry{
			Entry: "x31", Channel: "x15", Node: "x17",
			EnqueuedAt: instant(1), AvailableAt: instant(2), AgeSeconds: 240,
		},
		Lease: &ItemLease{ID: "x16", Channel: "x15", GrantedAt: instant(3), ExpiresAt: instant(4)},
		Need:  &NeedSnapshot{Event: "x77", Kind: "x12", Target: "x17", At: instant(5)},
	})
	if calls != 2 {
		t.Fatalf("query calls = %d, want 2", calls)
	}
}

func TestOperationsItemJournalScansMetadata(t *testing.T) {
	var gotArgs []any
	operations := &Operations{rows: func(_ context.Context, _ string, args ...any) (rowsScanner, error) {
		gotArgs = append([]any(nil), args...)
		return &rowsValues{rows: [][]any{{
			"x40",
			"x21",
			instant(0),
			int64(9),
			[]byte(`{"channel_key":"x15","routing_rule_order":2}`),
		}}}, nil
	}}

	got, err := operations.ItemJournal(context.Background(), JournalQuery{
		WorkItem:         "x08",
		Limit:            20,
		AfterAppendIndex: 8,
	})
	if err != nil {
		t.Fatal(err)
	}

	requireArgs(t, gotArgs, "x08", int64(8), 20)
	if len(got) != 1 {
		t.Fatalf("journal events = %+v, want one event", got)
	}
	requireJournalEvent(t, got[0], JournalEvent{Event: "x40", Kind: "x21", AppendedAt: instant(0), AppendIndex: 9})
	if got[0].Metadata["channel_key"] != "x15" || got[0].Metadata["routing_rule_order"] != float64(2) {
		t.Fatalf("metadata = %+v", got[0].Metadata)
	}
}

func TestOperationsNodesScansRoutabilityRoster(t *testing.T) {
	operations := &Operations{rows: func(context.Context, string, ...any) (rowsScanner, error) {
		return &rowsValues{rows: [][]any{
			{"x17", "x15", "x12,x13", false},
			{"x18", "x68", "x12", true},
		}}, nil
	}}

	got, err := operations.Nodes(context.Background(), NodeQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}

	want := []Node{
		{Key: "x17", Channel: "x15", NeedKinds: []string{"x12", "x13"}, Routable: false},
		{Key: "x18", Channel: "x68", NeedKinds: []string{"x12"}, Routable: true},
	}
	requireNodes(t, got, want)
}

func requireChannels(t *testing.T, got []ChannelSummary, want []ChannelSummary) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("channels = %+v, want %+v", got, want)
	}
	for index := range got {
		if got[index] != want[index] {
			t.Fatalf("channel[%d] = %+v, want %+v", index, got[index], want[index])
		}
	}
}

func requireChannelItems(t *testing.T, got []ChannelItem, want []ChannelItem) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("channel items = %+v, want %+v", got, want)
	}
	for index := range got {
		requireChannelItem(t, got[index], want[index])
	}
}

func requireChannelItem(t *testing.T, got ChannelItem, want ChannelItem) {
	t.Helper()

	if got.Entry != want.Entry || got.WorkItem != want.WorkItem || got.Channel != want.Channel || got.Node != want.Node {
		t.Fatalf("channel item = %+v, want %+v", got, want)
	}
	if got.EnqueuedAt != want.EnqueuedAt || got.AvailableAt != want.AvailableAt || got.AgeSeconds != want.AgeSeconds {
		t.Fatalf("channel item timing = %+v, want %+v", got, want)
	}
	requireLease(t, got.Lease, want.Lease)
}

func requireItem(t *testing.T, got ItemDetail, want ItemDetail) {
	t.Helper()

	if got.WorkItem != want.WorkItem || got.Kind != want.Kind {
		t.Fatalf("item = %+v, want %+v", got, want)
	}
	requireItemEntry(t, got.Entry, want.Entry)
	requireItemLease(t, got.Lease, want.Lease)
	requireNeed(t, got.Need, want.Need)
}

func requireItemEntry(t *testing.T, got *ItemEntry, want *ItemEntry) {
	t.Helper()

	if got == nil || want == nil {
		if got != want {
			t.Fatalf("item entry = %+v, want %+v", got, want)
		}
		return
	}
	if *got != *want {
		t.Fatalf("item entry = %+v, want %+v", got, want)
	}
}

func requireLease(t *testing.T, got *Lease, want *Lease) {
	t.Helper()

	if got == nil || want == nil {
		if got != want {
			t.Fatalf("lease = %+v, want %+v", got, want)
		}
		return
	}
	if *got != *want {
		t.Fatalf("lease = %+v, want %+v", got, want)
	}
}

func requireItemLease(t *testing.T, got *ItemLease, want *ItemLease) {
	t.Helper()

	if got == nil || want == nil {
		if got != want {
			t.Fatalf("item lease = %+v, want %+v", got, want)
		}
		return
	}
	if *got != *want {
		t.Fatalf("item lease = %+v, want %+v", got, want)
	}
}

func requireNeed(t *testing.T, got *NeedSnapshot, want *NeedSnapshot) {
	t.Helper()

	if got == nil || want == nil {
		if got != want {
			t.Fatalf("need = %+v, want %+v", got, want)
		}
		return
	}
	if *got != *want {
		t.Fatalf("need = %+v, want %+v", got, want)
	}
}

func requireJournalEvent(t *testing.T, got JournalEvent, want JournalEvent) {
	t.Helper()

	if got.Event != want.Event || got.Kind != want.Kind {
		t.Fatalf("journal event = %+v, want %+v", got, want)
	}
	if got.AppendedAt != want.AppendedAt || got.AppendIndex != want.AppendIndex {
		t.Fatalf("journal event timing = %+v, want %+v", got, want)
	}
}

func requireNodes(t *testing.T, got []Node, want []Node) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("nodes = %+v, want %+v", got, want)
	}
	for index := range got {
		requireNode(t, got[index], want[index])
	}
}

func requireNode(t *testing.T, got Node, want Node) {
	t.Helper()

	if got.Key != want.Key || got.Channel != want.Channel || got.Routable != want.Routable {
		t.Fatalf("node = %+v, want %+v", got, want)
	}
	if len(got.NeedKinds) != len(want.NeedKinds) {
		t.Fatalf("node need kinds = %+v, want %+v", got.NeedKinds, want.NeedKinds)
	}
	for index := range got.NeedKinds {
		if got.NeedKinds[index] != want.NeedKinds[index] {
			t.Fatalf("need kind[%d] = %q, want %q", index, got.NeedKinds[index], want.NeedKinds[index])
		}
	}
}
