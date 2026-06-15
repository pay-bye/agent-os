package readmodel

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestResponseComposesCompleteViews(t *testing.T) {
	model := New(
		WithClock(fixedClock{value: instant()}),
		WithPressureSource(fixedPressure{}),
		WithJournalSource(fixedJournal{}),
		WithCounterSource(fixedCounters{}),
		WithBuildSource(fixedBuild{}),
		WithCompatibilitySource(fixedCompatibility{}),
	)

	got := model.Response(context.Background())

	want := Response{
		GeneratedAt:   "2026-05-18T12:00:00Z",
		Result:        Complete,
		WindowSeconds: 300,
		Views: Views{
			Queue:         &Queue{ChannelClass: AllChannels, Depth: 12, Available: 7, OldestAvailableAgeSeconds: 84},
			Leases:        &Leases{ChannelClass: AllChannels, Held: 3, Expired: 1},
			Journal:       &Journal{AppendRatePerSecond: 0.4, WindowSeconds: 300},
			Commands:      &Commands{Succeeded: 40, Failed: 2},
			Routing:       &Routing{Routed: 38, Unrouted: 4},
			Build:         &Build{Version: "v1.2.3", Revision: "a1b2c3"},
			Compatibility: &Compatibility{ContractVersion: "v1", Features: []string{"x91"}, Routes: []Route{{Method: "GET", Path: "/x91"}}},
		},
		Unavailable: []Group{},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("response = %+v, want %+v", got, want)
	}
}

func TestResponseOmitsFailedGroupsWithoutCachedValues(t *testing.T) {
	pressure := &flippingPressure{}
	model := New(
		WithClock(fixedClock{value: instant()}),
		WithPressureSource(pressure),
		WithJournalSource(fixedJournal{}),
	)

	first := model.Response(context.Background())
	pressure.fail = true
	second := model.Response(context.Background())

	if first.Views.Queue == nil || first.Views.Leases == nil {
		t.Fatalf("first response missing pressure groups: %+v", first)
	}
	if second.Views.Queue != nil || second.Views.Leases != nil {
		t.Fatalf("second response reused cached pressure groups: %+v", second)
	}
	requireGroups(t, second.Unavailable, QueueGroup, LeasesGroup, CommandsGroup, RoutingGroup, BuildGroup, CompatibilityGroup)
}

func TestResponseMarksNoGroupsAvailable(t *testing.T) {
	model := New(
		WithClock(fixedClock{value: instant()}),
		WithPressureSource(failingPressure{}),
		WithJournalSource(failingJournal{}),
		WithCounterSource(failingCounters{}),
		WithBuildSource(failingBuild{}),
		WithCompatibilitySource(failingCompatibility{}),
	)

	got := model.Response(context.Background())

	if got.Result != Partial {
		t.Fatalf("result = %q, want partial", got.Result)
	}
	if got.Available() {
		t.Fatalf("available = true, want false: %+v", got)
	}
	requireGroups(t, got.Unavailable, QueueGroup, LeasesGroup, JournalGroup, CommandsGroup, RoutingGroup, BuildGroup, CompatibilityGroup)
}

func TestResponseDoesNotExposeDependencyErrorText(t *testing.T) {
	model := New(
		WithClock(fixedClock{value: instant()}),
		WithPressureSource(failingPressure{}),
		WithJournalSource(failingJournal{}),
		WithCounterSource(failingCounters{}),
		WithBuildSource(failingBuild{}),
		WithCompatibilitySource(failingCompatibility{}),
	)

	body, err := json.Marshal(model.Response(context.Background()))
	if err != nil {
		t.Fatal(err)
	}

	for _, value := range []string{"postgres://x91", "SELECT x91", "x91-secret-token", "x91-disposition"} {
		if strings.Contains(string(body), value) {
			t.Fatalf("response leaked %q: %s", value, body)
		}
	}
}

func instant() time.Time {
	return time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
}

type fixedClock struct {
	value time.Time
}

func (c fixedClock) Now() time.Time {
	return c.value
}

type fixedPressure struct{}

func (fixedPressure) Pressure(context.Context, time.Time) (Pressure, error) {
	return Pressure{
		Depth:                     12,
		Available:                 7,
		Held:                      3,
		Expired:                   1,
		OldestAvailableAgeSeconds: 84,
	}, nil
}

type flippingPressure struct {
	fail bool
}

func (p *flippingPressure) Pressure(context.Context, time.Time) (Pressure, error) {
	if p.fail {
		return Pressure{}, errors.New("postgres://x91")
	}
	return fixedPressure{}.Pressure(context.Background(), time.Time{})
}

type failingPressure struct{}

func (failingPressure) Pressure(context.Context, time.Time) (Pressure, error) {
	return Pressure{}, errors.New("postgres://x91")
}

type fixedJournal struct{}

func (fixedJournal) Journal(context.Context, time.Time, time.Duration) (Window, error) {
	return Window{Appends: 120, Seconds: 300}, nil
}

type failingJournal struct{}

func (failingJournal) Journal(context.Context, time.Time, time.Duration) (Window, error) {
	return Window{}, errors.New("SELECT x91")
}

type fixedCounters struct{}

func (fixedCounters) Counters(context.Context) (Counters, error) {
	return Counters{CommandsSucceeded: 40, CommandsFailed: 2, Routed: 38, Unrouted: 4}, nil
}

type failingCounters struct{}

func (failingCounters) Counters(context.Context) (Counters, error) {
	return Counters{}, errors.New("x91-route")
}

type fixedBuild struct{}

func (fixedBuild) Build(context.Context) (Build, error) {
	return Build{Version: "v1.2.3", Revision: "a1b2c3"}, nil
}

type failingBuild struct{}

func (failingBuild) Build(context.Context) (Build, error) {
	return Build{}, errors.New("x91-disposition")
}

type fixedCompatibility struct{}

func (fixedCompatibility) Compatibility(context.Context) (Compatibility, error) {
	return Compatibility{
		ContractVersion: "v1",
		Features:        []string{"x91"},
		Routes:          []Route{{Method: "GET", Path: "/x91"}},
	}, nil
}

type failingCompatibility struct{}

func (failingCompatibility) Compatibility(context.Context) (Compatibility, error) {
	return Compatibility{}, errors.New("x91-secret-token")
}

func requireGroups(t *testing.T, got []Group, want ...Group) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unavailable = %v, want %v", got, want)
	}
}
