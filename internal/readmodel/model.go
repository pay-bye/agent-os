package readmodel

import (
	"context"
	"time"
)

const defaultWindow = 5 * time.Minute

type Option func(*Model)

type Model struct {
	clock         Clock
	window        time.Duration
	channels      ChannelSource
	items         ChannelItemSource
	item          ItemSource
	itemJournal   ItemJournalSource
	nodes         NodeSource
	pressure      PressureSource
	journal       JournalSource
	counters      CounterSource
	build         BuildSource
	compatibility CompatibilitySource
}

func (m Model) Response(ctx context.Context) Response {
	now := m.clock.Now().UTC()
	response := newResponse(now, m.window)
	m.addPressure(ctx, now, &response)
	m.addJournal(ctx, now, &response)
	m.addCounters(ctx, &response)
	m.addBuild(ctx, &response)
	m.addCompatibility(ctx, &response)
	finish(&response)
	return response
}

func (m Model) addPressure(ctx context.Context, now time.Time, response *Response) {
	if m.pressure == nil {
		omit(response, QueueGroup, LeasesGroup)
		return
	}
	pressure, err := m.pressure.Pressure(ctx, now)
	if err != nil {
		omit(response, QueueGroup, LeasesGroup)
		return
	}
	response.Views.Queue = queue(pressure)
	response.Views.Leases = leases(pressure)
}

func (m Model) addJournal(ctx context.Context, now time.Time, response *Response) {
	if m.journal == nil {
		omit(response, JournalGroup)
		return
	}
	window, err := m.journal.Journal(ctx, now, m.window)
	if err != nil {
		omit(response, JournalGroup)
		return
	}
	response.Views.Journal = journal(window)
}

func (m Model) addCounters(ctx context.Context, response *Response) {
	if m.counters == nil {
		omit(response, CommandsGroup, RoutingGroup)
		return
	}
	counters, err := m.counters.Counters(ctx)
	if err != nil {
		omit(response, CommandsGroup, RoutingGroup)
		return
	}
	response.Views.Commands = commands(counters)
	response.Views.Routing = routing(counters)
}

func (m Model) addBuild(ctx context.Context, response *Response) {
	if m.build == nil {
		omit(response, BuildGroup)
		return
	}
	build, err := m.build.Build(ctx)
	if err != nil {
		omit(response, BuildGroup)
		return
	}
	response.Views.Build = &build
}

func (m Model) addCompatibility(ctx context.Context, response *Response) {
	if m.compatibility == nil {
		omit(response, CompatibilityGroup)
		return
	}
	compatibility, err := m.compatibility.Compatibility(ctx)
	if err != nil {
		omit(response, CompatibilityGroup)
		return
	}
	response.Views.Compatibility = &compatibility
}

func (m Model) Channels(ctx context.Context, query ChannelQuery) (ChannelList, error) {
	if m.channels == nil {
		return ChannelList{}, ErrUnavailable
	}
	channels, err := m.channels.Channels(ctx, m.clock.Now().UTC(), query)
	if err != nil {
		return ChannelList{}, err
	}
	return ChannelList{Channels: channels}, nil
}

func (m Model) ChannelItems(ctx context.Context, query ChannelItemQuery) (ChannelItemList, error) {
	if m.items == nil {
		return ChannelItemList{}, ErrUnavailable
	}
	items, err := m.items.ChannelItems(ctx, m.clock.Now().UTC(), query)
	if err != nil {
		return ChannelItemList{}, err
	}
	return ChannelItemList{Items: items}, nil
}

func (m Model) Item(ctx context.Context, id string) (ItemDetail, error) {
	if m.item == nil {
		return ItemDetail{}, ErrUnavailable
	}
	return m.item.Item(ctx, m.clock.Now().UTC(), id)
}

func (m Model) ItemJournal(ctx context.Context, query JournalQuery) (JournalEventList, error) {
	if m.itemJournal == nil {
		return JournalEventList{}, ErrUnavailable
	}
	events, err := m.itemJournal.ItemJournal(ctx, query)
	if err != nil {
		return JournalEventList{}, err
	}
	return JournalEventList{Events: events}, nil
}

func (m Model) Nodes(ctx context.Context, query NodeQuery) (NodeList, error) {
	if m.nodes == nil {
		return NodeList{}, ErrUnavailable
	}
	nodes, err := m.nodes.Nodes(ctx, query)
	if err != nil {
		return NodeList{}, err
	}
	return NodeList{Nodes: nodes}, nil
}

func New(options ...Option) Model {
	model := Model{
		clock:  systemClock{},
		window: defaultWindow,
	}
	for _, option := range options {
		option(&model)
	}
	return model
}

func WithClock(clock Clock) Option {
	return func(model *Model) {
		if clock != nil {
			model.clock = clock
		}
	}
}

func WithPressureSource(source PressureSource) Option {
	return func(model *Model) {
		model.pressure = source
	}
}

func WithJournalSource(source JournalSource) Option {
	return func(model *Model) {
		model.journal = source
	}
}

func WithCounterSource(source CounterSource) Option {
	return func(model *Model) {
		model.counters = source
	}
}

func WithChannelSource(source ChannelSource) Option {
	return func(model *Model) {
		model.channels = source
	}
}

func WithChannelItemSource(source ChannelItemSource) Option {
	return func(model *Model) {
		model.items = source
	}
}

func WithItemSource(source ItemSource) Option {
	return func(model *Model) {
		model.item = source
	}
}

func WithItemJournalSource(source ItemJournalSource) Option {
	return func(model *Model) {
		model.itemJournal = source
	}
}

func WithNodeSource(source NodeSource) Option {
	return func(model *Model) {
		model.nodes = source
	}
}

func WithBuildSource(source BuildSource) Option {
	return func(model *Model) {
		model.build = source
	}
}

func WithCompatibilitySource(source CompatibilitySource) Option {
	return func(model *Model) {
		model.compatibility = source
	}
}

func newResponse(now time.Time, window time.Duration) Response {
	return Response{
		GeneratedAt:   now.Format(time.RFC3339),
		WindowSeconds: int(window.Seconds()),
		Views:         Views{},
		Unavailable:   []Group{},
	}
}

func queue(pressure Pressure) *Queue {
	return &Queue{
		ChannelClass:              AllChannels,
		Depth:                     pressure.Depth,
		Available:                 pressure.Available,
		OldestAvailableAgeSeconds: pressure.OldestAvailableAgeSeconds,
	}
}

func leases(pressure Pressure) *Leases {
	return &Leases{
		ChannelClass: AllChannels,
		Held:         pressure.Held,
		Expired:      pressure.Expired,
	}
}

func journal(window Window) *Journal {
	return &Journal{
		AppendRatePerSecond: float64(window.Appends) / float64(window.Seconds),
		WindowSeconds:       window.Seconds,
	}
}

func commands(counters Counters) *Commands {
	return &Commands{
		Succeeded: counters.CommandsSucceeded,
		Failed:    counters.CommandsFailed,
	}
}

func routing(counters Counters) *Routing {
	return &Routing{
		Routed:   counters.Routed,
		Unrouted: counters.Unrouted,
	}
}

func omit(response *Response, groups ...Group) {
	response.Unavailable = append(response.Unavailable, groups...)
}

func finish(response *Response) {
	if len(response.Unavailable) == 0 {
		response.Result = Complete
		return
	}
	response.Result = Partial
}
