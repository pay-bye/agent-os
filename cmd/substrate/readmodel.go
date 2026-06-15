package main

import (
	"context"
	"database/sql"
	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/readmodel"
	readpostgres "github.com/pay-bye/agent-os/internal/readmodel/postgres"
	"github.com/pay-bye/agent-os/internal/transport/http/compatibility"
	"github.com/pay-bye/agent-os/internal/transport/http/operations"
	"time"
)

type pressureSource interface {
	Pressure(context.Context, time.Time) (readpostgres.Pressure, error)
}

type pressureReader struct {
	source pressureSource
}

func (r pressureReader) Pressure(ctx context.Context, now time.Time) (readmodel.Pressure, error) {
	pressure, err := r.source.Pressure(ctx, now)
	if err != nil {
		return readmodel.Pressure{}, err
	}
	return readmodel.Pressure{
		Depth:                     pressure.Depth,
		Available:                 pressure.Available,
		Held:                      pressure.Held,
		Expired:                   pressure.Expired,
		OldestAvailableAgeSeconds: pressure.OldestAvailableAgeSeconds,
	}, nil
}

type journalSource interface {
	Journal(context.Context, time.Time, time.Duration) (readpostgres.JournalWindow, error)
}

type journalReader struct {
	source journalSource
}

func (r journalReader) Journal(
	ctx context.Context,
	now time.Time,
	window time.Duration,
) (readmodel.Window, error) {
	item, err := r.source.Journal(ctx, now, window)
	if err != nil {
		return readmodel.Window{}, err
	}
	return readmodel.Window{Appends: item.Appends, Seconds: item.WindowSeconds}, nil
}

type counterReader struct {
	collector *metrics.Collector
}

func (r counterReader) Counters(context.Context) (readmodel.Counters, error) {
	snapshot := r.collector.CounterSnapshot()
	return readmodel.Counters{
		CommandsSucceeded: snapshot.Commands.Succeeded,
		CommandsFailed:    snapshot.Commands.Failed,
		Routed:            snapshot.Routing.Routed,
		Unrouted:          snapshot.Routing.Unrouted,
	}, nil
}

type channelSource interface {
	Channels(context.Context, time.Time, readmodel.ChannelQuery) ([]readmodel.Channel, error)
}

type channelReader struct {
	source channelSource
}

func (r channelReader) Channels(
	ctx context.Context,
	now time.Time,
	query readmodel.ChannelQuery,
) ([]readmodel.Channel, error) {
	return r.source.Channels(ctx, now, query)
}

type channelItemSource interface {
	ChannelItems(context.Context, time.Time, readmodel.ChannelItemQuery) ([]readmodel.ChannelItem, error)
}

type channelItemReader struct {
	source channelItemSource
}

func (r channelItemReader) ChannelItems(
	ctx context.Context,
	now time.Time,
	query readmodel.ChannelItemQuery,
) ([]readmodel.ChannelItem, error) {
	return r.source.ChannelItems(ctx, now, query)
}

type itemSource interface {
	Item(context.Context, time.Time, string) (readmodel.ItemDetail, error)
}

type itemReader struct {
	source itemSource
}

func (r itemReader) Item(ctx context.Context, now time.Time, id string) (readmodel.ItemDetail, error) {
	item, err := r.source.Item(ctx, now, id)
	if err != nil {
		return readmodel.ItemDetail{}, err
	}
	return item, nil
}

type itemJournalSource interface {
	ItemJournal(context.Context, readmodel.JournalQuery) ([]readmodel.JournalEvent, error)
}

type itemJournalReader struct {
	source itemJournalSource
}

func (r itemJournalReader) ItemJournal(
	ctx context.Context,
	query readmodel.JournalQuery,
) ([]readmodel.JournalEvent, error) {
	return r.source.ItemJournal(ctx, query)
}

type nodeSource interface {
	Nodes(context.Context, readmodel.NodeQuery) ([]readmodel.Node, error)
}

type nodeReader struct {
	source nodeSource
}

func (r nodeReader) Nodes(ctx context.Context, query readmodel.NodeQuery) ([]readmodel.Node, error) {
	return r.source.Nodes(ctx, query)
}

type buildReader struct {
	collector *metrics.Collector
}

func (r buildReader) Build(context.Context) (readmodel.Build, error) {
	build := r.collector.CounterSnapshot().Build
	return readmodel.Build{Version: build.Version, Revision: build.Revision}, nil
}

type compatibilityReader struct{}

func (compatibilityReader) Compatibility(context.Context) (readmodel.Compatibility, error) {
	contract := compatibility.CompatibilityContract()
	return readmodel.Compatibility{
		ContractVersion: contract.ContractVersion,
		Features:        append([]string{}, contract.Features...),
		Routes:          compatibilityRoutes(contract.Routes),
	}, nil
}

type operationsView struct {
	model readmodel.Model
}

func (v operationsView) Response(ctx context.Context) operations.OperationsReport {
	return v.model.Response(ctx)
}

func (v operationsView) Channels(ctx context.Context, query operations.ChannelQuery) (any, error) {
	return v.model.Channels(ctx, readmodel.ChannelQuery{
		Limit:            query.Limit,
		After:            query.After,
		OlderThanSeconds: query.OlderThanSeconds,
	})
}

func (v operationsView) ChannelItems(ctx context.Context, query operations.ChannelItemQuery) (any, error) {
	return v.model.ChannelItems(ctx, readmodel.ChannelItemQuery{
		Channel:          query.Channel,
		Limit:            query.Limit,
		OlderThanSeconds: query.OlderThanSeconds,
		Lease:            query.Lease,
	})
}

func (v operationsView) Item(ctx context.Context, id string) (any, error) {
	return v.model.Item(ctx, id)
}

func (v operationsView) ItemJournal(ctx context.Context, query operations.ItemJournalQuery) (any, error) {
	return v.model.ItemJournal(ctx, readmodel.JournalQuery{
		WorkItem:         query.WorkItem,
		Limit:            query.Limit,
		AfterAppendIndex: query.AfterAppendIndex,
	})
}

func (v operationsView) Nodes(ctx context.Context, query operations.NodeQuery) (any, error) {
	return v.model.Nodes(ctx, readmodel.NodeQuery{
		Limit:    query.Limit,
		After:    query.After,
		NeedKind: query.NeedKind,
	})
}

func newOperationsView(
	db *sql.DB,
	collector *metrics.Collector,
) operations.Operations {
	store := readpostgres.NewOperations(db)
	return operationsView{model: readmodel.New(
		readmodel.WithChannelSource(channelReader{source: store}),
		readmodel.WithChannelItemSource(channelItemReader{source: store}),
		readmodel.WithItemSource(itemReader{source: store}),
		readmodel.WithItemJournalSource(itemJournalReader{source: store}),
		readmodel.WithNodeSource(nodeReader{source: store}),
		readmodel.WithPressureSource(pressureReader{source: store}),
		readmodel.WithJournalSource(journalReader{source: store}),
		readmodel.WithCounterSource(counterReader{collector: collector}),
		readmodel.WithBuildSource(buildReader{collector: collector}),
		readmodel.WithCompatibilitySource(compatibilityReader{}),
	)}
}

func compatibilityRoutes(input []compatibility.CompatibilityRoute) []readmodel.Route {
	output := make([]readmodel.Route, 0, len(input))
	for _, item := range input {
		output = append(output, readmodel.Route{Method: item.Method, Path: item.Path})
	}
	return output
}
