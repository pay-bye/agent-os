package main

import (
	"context"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/readmodel"
	readpostgres "github.com/pay-bye/agent-os/internal/readmodel/postgres"
)

func TestPressureReaderMapsStorageValues(t *testing.T) {
	reader := pressureReader{source: fixedPressureSource{}}

	got, err := reader.Pressure(context.Background(), readInstant())
	if err != nil {
		t.Fatal(err)
	}

	if got.Depth != 12 || got.Available != 7 {
		t.Fatalf("queue pressure = %+v", got)
	}
	if got.Held != 3 || got.Expired != 1 {
		t.Fatalf("lease pressure = %+v", got)
	}
	if got.OldestAvailableAgeSeconds != 84 {
		t.Fatalf("oldest age = %d, want 84", got.OldestAvailableAgeSeconds)
	}
}

func TestJournalReaderMapsStorageWindow(t *testing.T) {
	reader := journalReader{source: fixedJournalSource{}}

	got, err := reader.Journal(context.Background(), readInstant(), 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	if got.Appends != 120 {
		t.Fatalf("appends = %d, want 120", got.Appends)
	}
	if got.Seconds != 300 {
		t.Fatalf("seconds = %d, want 300", got.Seconds)
	}
}

func TestCounterAndBuildReadersUseCollectorSnapshot(t *testing.T) {
	collector := metrics.New(metrics.WithBuild(metrics.Build{Version: "v1.2.3", Revision: "a1b2c3"}))
	collector.ObserveRequest(metrics.Submit, metrics.Completed, time.Millisecond)
	collector.ObserveRequest(metrics.Claim, metrics.Failed, time.Millisecond)
	collector.ObserveRouting(metrics.Routed)
	collector.ObserveRouting(metrics.Unrouted)

	counters, err := counterReader{collector: collector}.Counters(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	build, err := buildReader{collector: collector}.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if counters.CommandsSucceeded != 1 || counters.CommandsFailed != 1 {
		t.Fatalf("commands = %+v", counters)
	}
	if counters.Routed != 1 || counters.Unrouted != 1 {
		t.Fatalf("routing = %+v", counters)
	}
	if build.Version != "v1.2.3" || build.Revision != "a1b2c3" {
		t.Fatalf("build = %+v", build)
	}
}

func TestLocatorReadersMapStorageValues(t *testing.T) {
	ctx := context.Background()
	channels, err := channelReader{source: fixedChannelSource{}}.Channels(ctx, readInstant(), readmodel.ChannelQuery{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	items, err := channelItemReader{source: fixedItemSource{}}.ChannelItems(ctx, readInstant(), readmodel.ChannelItemQuery{Channel: "x15"})
	if err != nil {
		t.Fatal(err)
	}
	item, err := itemReader{source: fixedDetailSource{}}.Item(ctx, readInstant(), "x08")
	if err != nil {
		t.Fatal(err)
	}
	events, err := itemJournalReader{source: fixedEventSource{}}.ItemJournal(ctx, readmodel.JournalQuery{WorkItem: "x08"})
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := nodeReader{source: fixedNodeSource{}}.Nodes(ctx, readmodel.NodeQuery{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}

	if channels[0].Key != "x15" || channels[0].Node != "x17" {
		t.Fatalf("channel = %+v, want x15/x17", channels[0])
	}
	if items[0].Lease == nil || items[0].Lease.ID != "x13" {
		t.Fatalf("item lease = %+v, want x13", items[0].Lease)
	}
	if item.Need == nil || item.Need.Kind != "x12" {
		t.Fatalf("need = %+v, want x12", item.Need)
	}
	if events[0].Metadata["need_kind"] != "x12" {
		t.Fatalf("metadata = %+v, want need kind", events[0].Metadata)
	}
	if nodes[0].Channel != "x15" || !equalStrings(nodes[0].NeedKinds, []string{"x12"}) {
		t.Fatalf("node = %+v, want channel and capability", nodes[0])
	}
}

type fixedPressureSource struct{}

func (fixedPressureSource) Pressure(context.Context, time.Time) (readpostgres.Pressure, error) {
	return readpostgres.Pressure{
		Depth:                     12,
		Available:                 7,
		Held:                      3,
		Expired:                   1,
		OldestAvailableAgeSeconds: 84,
	}, nil
}

type fixedJournalSource struct{}

func (fixedJournalSource) Journal(context.Context, time.Time, time.Duration) (readpostgres.JournalWindow, error) {
	return readpostgres.JournalWindow{Appends: 120, WindowSeconds: 300}, nil
}

func readInstant() time.Time {
	return time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
}

type fixedChannelSource struct{}

func (fixedChannelSource) Channels(context.Context, time.Time, readmodel.ChannelQuery) ([]readmodel.Channel, error) {
	return []readmodel.Channel{{
		Key:                       "x15",
		Node:                      "x17",
		Depth:                     2,
		Available:                 1,
		OldestAvailableAgeSeconds: 60,
	}}, nil
}

type fixedItemSource struct{}

func (fixedItemSource) ChannelItems(
	context.Context,
	time.Time,
	readmodel.ChannelItemQuery,
) ([]readmodel.ChannelItem, error) {
	return []readmodel.ChannelItem{{
		Entry:       "x01",
		WorkItem:    "x08",
		Channel:     "x15",
		Node:        "x17",
		EnqueuedAt:  readInstant().Add(-2 * time.Minute),
		AvailableAt: readInstant().Add(-time.Minute),
		AgeSeconds:  60,
		Lease:       &readmodel.Lease{ID: "x13", GrantedAt: readInstant(), ExpiresAt: readInstant().Add(time.Minute)},
	}}, nil
}

type fixedDetailSource struct{}

func (fixedDetailSource) Item(context.Context, time.Time, string) (readmodel.ItemDetail, error) {
	return readmodel.ItemDetail{
		WorkItem:    "x08",
		Kind:        "x03",
		SubmittedAt: readInstant().Add(-5 * time.Minute),
		Need:        &readmodel.NeedSnapshot{Event: "x21", Kind: "x12", Target: "x17", At: readInstant()},
	}, nil
}

type fixedEventSource struct{}

func (fixedEventSource) ItemJournal(context.Context, readmodel.JournalQuery) ([]readmodel.JournalEvent, error) {
	return []readmodel.JournalEvent{{
		Event:       "x21",
		Kind:        "x41",
		AppendedAt:  readInstant(),
		AppendIndex: 1,
		Metadata:    map[string]any{"need_kind": "x12"},
	}}, nil
}

type fixedNodeSource struct{}

func (fixedNodeSource) Nodes(context.Context, readmodel.NodeQuery) ([]readmodel.Node, error) {
	return []readmodel.Node{{Key: "x17", Channel: "x15", NeedKinds: []string{"x12"}, Routable: true}}, nil
}

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
