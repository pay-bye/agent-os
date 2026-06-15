package metrics

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCollectorEmitsBoundedText(t *testing.T) {
	store := fixedStore{values: Storage{
		QueueDepth:    3,
		LeasesHeld:    2,
		LeasesExpired: 1,
	}}
	collector := New(
		WithClock(fixedClock{value: time.Unix(1_779_000_000, 0)}),
		WithBuild(Build{Version: "v1.2.3", Revision: "a1b2c3"}),
		WithStore(store),
	)

	collector.ObserveRequest(Submit, Completed, 25*time.Millisecond)
	collector.ObserveAuthRejection()
	collector.ObserveJournalAppend(ItemSubmitted)
	collector.ObserveRouting(Routed)
	collector.ObserveDeclaration(Preview, Succeeded, 10*time.Millisecond)
	collector.ObserveMigration(Succeeded, 50*time.Millisecond)

	text := collector.Text(context.Background())

	requireLine(t, text, "# HELP requests_total Total credential-gated HTTP requests by bounded operation, result, and protocol.")
	requireLine(t, text, "# TYPE request_duration_seconds histogram")
	requireLine(t, text, `requests_total{operation="submit",result="completed",protocol="http"} 1`)
	requireLine(t, text, `auth_rejections_total{family="unauthorized",protocol="http"} 1`)
	requireLine(t, text, `queue_depth{channel_class="all"} 3`)
	requireLine(t, text, `leases_held{channel_class="all"} 2`)
	requireLine(t, text, `leases_expired{channel_class="all"} 1`)
	requireLine(t, text, `journal_appends_total{event_kind="item_submitted"} 1`)
	requireLine(t, text, `routing_results_total{outcome="routed"} 1`)
	requireLine(t, text, `declaration_operations_total{operation="preview",result="succeeded"} 1`)
	requireLine(t, text, `migrations_total{result="succeeded"} 1`)
	requireLine(t, text, `process_start_time_seconds 1779000000`)
	requireLine(t, text, `build_info{version="v1.2.3",revision="a1b2c3"} 1`)
}

func TestCollectorRejectsRawLabelValues(t *testing.T) {
	collector := New(WithBuild(Build{
		Version:  `x/y`,
		Revision: `x y`,
	}))

	collector.ObserveRequest(Operation("opaque-x99"), Result("opaque-y77"), time.Millisecond)
	collector.ObserveJournalAppend(EventKind("x40"))
	collector.ObserveRouting(Outcome("opaque-z15"))

	text := collector.Text(context.Background())

	rejectText(t, text, "opaque-x99")
	rejectText(t, text, "opaque-y77")
	rejectText(t, text, "opaque-z15")
	rejectText(t, text, "x/y")
	rejectText(t, text, "x y")
	requireLine(t, text, `build_info{version="unknown",revision="unknown"} 1`)
}

func TestCollectorRejectsInvalidBuildLabels(t *testing.T) {
	for _, test := range rejectedBuildLabelCases() {
		t.Run(test.name, func(t *testing.T) {
			collector := New(WithBuild(Build{
				Version:  test.value,
				Revision: test.value,
			}))

			text := collector.Text(context.Background())

			rejectText(t, text, test.value)
			requireLine(t, text, `build_info{version="unknown",revision="unknown"} 1`)
		})
	}
}

func TestCollectorCounterSnapshotReturnsBoundedReadValues(t *testing.T) {
	collector := New(WithBuild(Build{Version: "v1.2.3", Revision: "a1b2c3"}))

	collector.ObserveRequest(Submit, Completed, time.Millisecond)
	collector.ObserveRequest(Claim, Failed, time.Millisecond)
	collector.ObserveRequest(Health, Completed, time.Millisecond)
	collector.ObserveRequest(Submit, Rejected, time.Millisecond)
	collector.ObserveRouting(Routed)
	collector.ObserveRouting(Unrouted)
	collector.ObserveRouting(NoRoute)

	got := collector.CounterSnapshot()

	if got.Commands.Succeeded != 1 {
		t.Fatalf("commands succeeded = %d, want 1", got.Commands.Succeeded)
	}
	if got.Commands.Failed != 1 {
		t.Fatalf("commands failed = %d, want 1", got.Commands.Failed)
	}
	if got.Routing.Routed != 1 {
		t.Fatalf("routing routed = %d, want 1", got.Routing.Routed)
	}
	if got.Routing.Unrouted != 1 {
		t.Fatalf("routing unrouted = %d, want 1", got.Routing.Unrouted)
	}
	if got.Build.Version != "v1.2.3" || got.Build.Revision != "a1b2c3" {
		t.Fatalf("build = %+v, want bounded build metadata", got.Build)
	}
}

func TestCollectorAcceptsBoundedBuildLabels(t *testing.T) {
	for _, test := range acceptedBuildLabelCases() {
		t.Run(test.name, func(t *testing.T) {
			collector := New(WithBuild(Build{
				Version:  test.version,
				Revision: test.revision,
			}))

			text := collector.Text(context.Background())

			line := `build_info{version="` + test.version + `",revision="` + test.revision + `"} 1`
			requireLine(t, text, line)
		})
	}
}

func TestCollectorOmitsStorageGaugesWhenStoreFails(t *testing.T) {
	collector := New(WithStore(failingStore{}))

	text := collector.Text(context.Background())

	rejectText(t, text, "queue_depth")
	rejectText(t, text, "leases_held")
	rejectText(t, text, "leases_expired")
}

func TestClosedSeriesCountStaysWithinBudget(t *testing.T) {
	if count := ClosedSeriesCount(); count > SeriesBudget {
		t.Fatalf("series count = %d, budget = %d", count, SeriesBudget)
	}
}

type fixedClock struct {
	value time.Time
}

func (c fixedClock) Now() time.Time {
	return c.value
}

type fixedStore struct {
	values Storage
}

func (s fixedStore) Read(context.Context, time.Time) (Storage, error) {
	return s.values, nil
}

type failingStore struct{}

func (failingStore) Read(context.Context, time.Time) (Storage, error) {
	return Storage{}, errors.New("postgres://secret raw failure")
}

type rejectedBuildLabelCase struct {
	name  string
	value string
}

func rejectedBuildLabelCases() []rejectedBuildLabelCase {
	return []rejectedBuildLabelCase{
		{name: "missing prefix", value: "1.2.3"},
		{name: "missing patch", value: "v1.2"},
		{name: "extra part", value: "v1.2.3.4"},
		{name: "non rc suffix", value: "v1.2.3-x.1"},
		{name: "suffix missing number", value: "v1.2.3-rc."},
		{name: "suffix with letters", value: "v1.2.3-rc.x"},
		{name: "underscore", value: "x_y"},
		{name: "path", value: "x/y"},
		{name: "whitespace", value: "x y"},
		{name: "quote", value: `v1"2`},
		{name: "backslash", value: `a\b`},
		{name: "uppercase hex", value: "ABCDEF"},
		{name: "too short hex", value: "abcde"},
		{name: "too long hex", value: "abcdefabcdefabcdefabcdefabcdefabcdefabcdef"},
	}
}

type acceptedBuildLabelCase struct {
	name     string
	version  string
	revision string
}

func acceptedBuildLabelCases() []acceptedBuildLabelCase {
	return []acceptedBuildLabelCase{
		{name: "release", version: "v10.24.7", revision: "abcdef1234567890"},
		{name: "release candidate", version: "v10.24.7-rc.1", revision: "0123456789abcdef"},
	}
}

func requireLine(t *testing.T, text string, line string) {
	t.Helper()

	if !strings.Contains(text, line+"\n") {
		t.Fatalf("metrics text missing %q:\n%s", line, text)
	}
}

func rejectText(t *testing.T, text string, value string) {
	t.Helper()

	if strings.Contains(text, value) {
		t.Fatalf("metrics text contains %q:\n%s", value, text)
	}
}
