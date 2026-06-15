package readmodel

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestSurfacesReturnDedicatedViews(t *testing.T) {
	source := &surfaceSource{}
	model := New(
		WithClock(fixedClock{value: instant()}),
		WithChannelSource(source),
		WithChannelItemSource(source),
		WithItemSource(source),
		WithItemJournalSource(source),
		WithNodeSource(source),
	)

	requireNoError(t, checkChannels(model, source))
	requireNoError(t, checkChannelItems(model, source))
	requireNoError(t, checkItem(model, source))
	requireNoError(t, checkItemJournal(model, source))
	requireNoError(t, checkNodes(model, source))
}

func TestSurfacesReturnUnavailableWithoutSources(t *testing.T) {
	model := New(WithClock(fixedClock{value: instant()}))

	checks := []error{
		channelError(model),
		channelItemError(model),
		itemError(model),
		itemJournalError(model),
		nodeError(model),
	}
	for _, err := range checks {
		if !errors.Is(err, ErrUnavailable) {
			t.Fatalf("error = %v, want %v", err, ErrUnavailable)
		}
	}
}

func checkChannels(model Model, source *surfaceSource) error {
	query := ChannelQuery{Limit: 50, After: "x14", OlderThanSeconds: 60}
	got, err := model.Channels(context.Background(), query)
	if err != nil {
		return err
	}
	want := ChannelList{Channels: []Channel{{Key: "x15", Node: "x17", Depth: 4, Available: 2, OldestAvailableAgeSeconds: 90}}}
	if !reflect.DeepEqual(got, want) {
		return errors.New("channels mismatch")
	}
	if source.channelNow != instant() || source.channelQuery != query {
		return errors.New("channel query mismatch")
	}
	return nil
}

func checkChannelItems(model Model, source *surfaceSource) error {
	query := ChannelItemQuery{Channel: "x15", Limit: 10, OlderThanSeconds: 30, Lease: "held"}
	got, err := model.ChannelItems(context.Background(), query)
	if err != nil {
		return err
	}
	want := ChannelItemList{Items: []ChannelItem{channelItem()}}
	if !reflect.DeepEqual(got, want) {
		return errors.New("channel items mismatch")
	}
	if source.itemNow != instant() || source.itemQuery != query {
		return errors.New("channel item query mismatch")
	}
	return nil
}

func checkItem(model Model, source *surfaceSource) error {
	got, err := model.Item(context.Background(), "x08")
	if err != nil {
		return err
	}
	want := itemDetail()
	if !reflect.DeepEqual(got, want) {
		return errors.New("item mismatch")
	}
	if source.detailNow != instant() || source.detailID != "x08" {
		return errors.New("item query mismatch")
	}
	return nil
}

func checkItemJournal(model Model, source *surfaceSource) error {
	query := JournalQuery{WorkItem: "x08", Limit: 20, AfterAppendIndex: 8}
	got, err := model.ItemJournal(context.Background(), query)
	if err != nil {
		return err
	}
	want := JournalEventList{Events: []JournalEvent{journalEvent()}}
	if !reflect.DeepEqual(got, want) {
		return errors.New("journal mismatch")
	}
	if source.journalQuery != query {
		return errors.New("journal query mismatch")
	}
	return nil
}

func checkNodes(model Model, source *surfaceSource) error {
	query := NodeQuery{Limit: 100, After: "x16", NeedKind: "x12"}
	got, err := model.Nodes(context.Background(), query)
	if err != nil {
		return err
	}
	want := NodeList{Nodes: []Node{{Key: "x17", Channel: "x15", NeedKinds: []string{"x12"}, Routable: true}}}
	if !reflect.DeepEqual(got, want) {
		return errors.New("nodes mismatch")
	}
	if source.nodeQuery != query {
		return errors.New("node query mismatch")
	}
	return nil
}

func channelError(model Model) error {
	_, err := model.Channels(context.Background(), ChannelQuery{})
	return err
}

func channelItemError(model Model) error {
	_, err := model.ChannelItems(context.Background(), ChannelItemQuery{})
	return err
}

func itemError(model Model) error {
	_, err := model.Item(context.Background(), "x08")
	return err
}

func itemJournalError(model Model) error {
	_, err := model.ItemJournal(context.Background(), JournalQuery{})
	return err
}

func nodeError(model Model) error {
	_, err := model.Nodes(context.Background(), NodeQuery{})
	return err
}

func channelItem() ChannelItem {
	return ChannelItem{
		Entry:       "x31",
		WorkItem:    "x08",
		Channel:     "x15",
		Node:        "x17",
		EnqueuedAt:  instant().Add(-2 * time.Minute),
		AvailableAt: instant().Add(-1 * time.Minute),
		AgeSeconds:  60,
		Lease:       &Lease{ID: "x16", GrantedAt: instant(), ExpiresAt: instant().Add(time.Minute)},
	}
}

func itemDetail() ItemDetail {
	return ItemDetail{
		WorkItem:    "x08",
		Kind:        "x91",
		SubmittedAt: instant(),
		Entry:       &ItemEntry{Entry: "x31", Channel: "x15", Node: "x17"},
		Lease:       &ItemLease{ID: "x16", Channel: "x15", GrantedAt: instant(), ExpiresAt: instant().Add(time.Minute)},
		Need:        &NeedSnapshot{Event: "x77", Kind: "x12", Target: "x17", At: instant()},
	}
}

func journalEvent() JournalEvent {
	return JournalEvent{
		Event:       "x40",
		Kind:        "x21",
		AppendedAt:  instant(),
		AppendIndex: 9,
		Metadata:    map[string]any{"channel_key": "x15"},
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}

type surfaceSource struct {
	channelNow   time.Time
	channelQuery ChannelQuery
	itemNow      time.Time
	itemQuery    ChannelItemQuery
	detailNow    time.Time
	detailID     string
	journalQuery JournalQuery
	nodeQuery    NodeQuery
}

func (s *surfaceSource) Channels(_ context.Context, now time.Time, query ChannelQuery) ([]Channel, error) {
	s.channelNow = now
	s.channelQuery = query
	return []Channel{{Key: "x15", Node: "x17", Depth: 4, Available: 2, OldestAvailableAgeSeconds: 90}}, nil
}

func (s *surfaceSource) ChannelItems(_ context.Context, now time.Time, query ChannelItemQuery) ([]ChannelItem, error) {
	s.itemNow = now
	s.itemQuery = query
	return []ChannelItem{channelItem()}, nil
}

func (s *surfaceSource) Item(_ context.Context, now time.Time, id string) (ItemDetail, error) {
	s.detailNow = now
	s.detailID = id
	return itemDetail(), nil
}

func (s *surfaceSource) ItemJournal(_ context.Context, query JournalQuery) ([]JournalEvent, error) {
	s.journalQuery = query
	return []JournalEvent{journalEvent()}, nil
}

func (s *surfaceSource) Nodes(_ context.Context, query NodeQuery) ([]Node, error) {
	s.nodeQuery = query
	return []Node{{Key: "x17", Channel: "x15", NeedKinds: []string{"x12"}, Routable: true}}, nil
}
