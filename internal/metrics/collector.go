package metrics

import (
	"context"
	"sync"
	"time"
)

type Option func(*Collector)

type requestKey struct {
	operation Operation
	result    Result
}

type requestDurationKey struct {
	operation Operation
	result    Result
	bucket    string
}

type declarationKey struct {
	operation DeclarationOperation
	result    Result
}

type declarationDurationKey struct {
	operation DeclarationOperation
	result    Result
	bucket    string
}

type migrationDurationKey struct {
	result Result
	bucket string
}

type Collector struct {
	lock sync.Mutex

	clock Clock
	build Build
	store Store
	start time.Time

	requests             map[requestKey]int
	requestDurations     map[requestDurationKey]int
	authRejections       int
	journalAppends       map[EventKind]int
	routingResults       map[Outcome]int
	declarations         map[declarationKey]int
	declarationDurations map[declarationDurationKey]int
	migrations           map[Result]int
	migrationDurations   map[migrationDurationKey]int
}

func (c *Collector) ObserveRequest(operation Operation, result Result, duration time.Duration) {
	if !validRequest(operation, result) {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	c.requests[requestKey{operation: operation, result: result}]++
	for _, bucket := range buckets() {
		if duration.Seconds() <= bucket.value {
			key := requestDurationKey{operation: operation, result: result, bucket: bucket.label}
			c.requestDurations[key]++
		}
	}
}

func (c *Collector) ObserveAuthRejection() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.authRejections++
}

func (c *Collector) ObserveJournalAppend(kind EventKind) {
	if !validEventKind(kind) {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	c.journalAppends[kind]++
}

func (c *Collector) ObserveRouting(outcome Outcome) {
	if !validOutcome(outcome) {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	c.routingResults[outcome]++
}

func (c *Collector) ObserveDeclaration(operation DeclarationOperation, result Result, duration time.Duration) {
	if !validDeclaration(operation, result) {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	c.declarations[declarationKey{operation: operation, result: result}]++
	for _, bucket := range buckets() {
		if duration.Seconds() <= bucket.value {
			key := declarationDurationKey{operation: operation, result: result, bucket: bucket.label}
			c.declarationDurations[key]++
		}
	}
}

func (c *Collector) ObserveMigration(result Result, duration time.Duration) {
	if !validOperationResult(result) {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	c.migrations[result]++
	for _, bucket := range buckets() {
		if duration.Seconds() <= bucket.value {
			c.migrationDurations[migrationDurationKey{result: result, bucket: bucket.label}]++
		}
	}
}

func (c *Collector) Text(ctx context.Context) string {
	c.lock.Lock()
	snapshot := c.snapshot()
	c.lock.Unlock()

	return encode(withStorage(ctx, snapshot))
}

func (c *Collector) snapshot() snapshot {
	return snapshot{
		start:                c.start,
		build:                c.build,
		store:                c.store,
		now:                  c.clock.Now(),
		requests:             cloneMap(c.requests),
		requestDurations:     cloneMap(c.requestDurations),
		authRejections:       c.authRejections,
		journalAppends:       cloneMap(c.journalAppends),
		routingResults:       cloneMap(c.routingResults),
		declarations:         cloneMap(c.declarations),
		declarationDurations: cloneMap(c.declarationDurations),
		migrations:           cloneMap(c.migrations),
		migrationDurations:   cloneMap(c.migrationDurations),
	}
}

func (c *Collector) CounterSnapshot() CounterSnapshot {
	c.lock.Lock()
	snapshot := c.snapshot()
	c.lock.Unlock()

	return CounterSnapshot{
		Commands: commandCounts(snapshot.requests),
		Routing: RoutingCounts{
			Routed:   snapshot.routingResults[Routed],
			Unrouted: snapshot.routingResults[Unrouted],
		},
		Build: snapshot.build,
	}
}

func New(options ...Option) *Collector {
	collector := &Collector{
		clock:                clock{},
		build:                Build{Version: "unknown", Revision: "unknown"},
		requests:             map[requestKey]int{},
		requestDurations:     map[requestDurationKey]int{},
		journalAppends:       map[EventKind]int{},
		routingResults:       map[Outcome]int{},
		declarations:         map[declarationKey]int{},
		declarationDurations: map[declarationDurationKey]int{},
		migrations:           map[Result]int{},
		migrationDurations:   map[migrationDurationKey]int{},
	}
	for _, option := range options {
		option(collector)
	}
	collector.start = collector.clock.Now()
	return collector
}

func WithClock(clock Clock) Option {
	return func(collector *Collector) {
		if clock != nil {
			collector.clock = clock
		}
	}
}

func WithBuild(build Build) Option {
	return func(collector *Collector) {
		collector.build = Build{
			Version:  buildVersionValue(build.Version),
			Revision: buildRevisionValue(build.Revision),
		}
	}
}

func WithStore(store Store) Option {
	return func(collector *Collector) {
		collector.store = store
	}
}

func cloneMap[K comparable](input map[K]int) map[K]int {
	output := make(map[K]int, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
