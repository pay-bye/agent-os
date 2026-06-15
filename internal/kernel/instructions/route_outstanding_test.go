package instructions

import (
	"context"
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/kernel/routing"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestRouteOutstandingCommandBuildsStoreCommand(t *testing.T) {
	command, err := RouteOutstanding(ItemsInput{
		ID:        "x70",
		WorkItems: []workitem.ID{"x08", "x09"},
	}, fixedClock{now: instant(0)}, &sequenceIDs{
		values: []string{"x80", "x81", "x82", "x83", "x31", "x32"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if command.Record.Kind != "route_outstanding" {
		t.Fatalf("kind = %q, want route_outstanding", command.Record.Kind)
	}
	if command.RouteEvents[0] != journal.EventID("x82") || command.RouteEvents[1] != journal.EventID("x83") {
		t.Fatalf("route events = %v, want x82/x83", command.RouteEvents)
	}
	if command.Entries[0] != channel.EntryID("x31") || command.Entries[1] != channel.EntryID("x32") {
		t.Fatalf("entries = %v, want x31/x32", command.Entries)
	}
}

func TestApplyRouteOutstandingBuildsRoutePlan(t *testing.T) {
	command := ItemsCommand{
		Record:      recordAt("x70"),
		WorkItems:   []workitem.ID{"x08"},
		Events:      []journal.EventID{"x80"},
		RouteEvents: []journal.EventID{"x81"},
		Entries:     []channel.EntryID{"x31"},
	}
	facts := RouteFacts{WorkItems: []RoutableWorkItem{{
		WorkItem: WorkItemState{ID: "x08", Exists: true},
		Need:     routing.Need{Kind: "x91"},
		NeedOpen: true,
	}}}

	application, err := ApplyRouteOutstanding(context.Background(), command, facts, staticRoutes{candidate: routeCandidate(t)})

	if err != nil {
		t.Fatal(err)
	}
	requireResult(t, application.Result, Applied, "")
	if len(application.Routes) != 1 || application.Routes[0].Effect.Selection.Node.Key() != "x17" {
		t.Fatalf("routes = %+v", application.Routes)
	}
}

func TestApplyRouteOutstandingRejectsMissingNeed(t *testing.T) {
	command := ItemsCommand{Record: recordAt("x70"), WorkItems: []workitem.ID{"x08"}, Events: []journal.EventID{"x80"}}
	facts := RouteFacts{WorkItems: []RoutableWorkItem{{WorkItem: WorkItemState{ID: "x08", Exists: true}}}}

	application, err := ApplyRouteOutstanding(context.Background(), command, facts, staticRoutes{})

	if err != nil {
		t.Fatal(err)
	}
	requireResult(t, application.Result, PreconditionFailed, "need_outstanding")
}

func routeCandidate(t *testing.T) routing.Candidate {
	t.Helper()

	return routing.Candidate{Found: true, Node: node(t, "x17", "x15", "x91")}
}

type staticRoutes struct {
	candidate routing.Candidate
}

func (r staticRoutes) AddressedTarget(context.Context, routing.Need) (routing.Candidate, error) {
	return r.candidate, nil
}

func (r staticRoutes) DefaultCandidates(context.Context, routing.Need) ([]routing.Candidate, error) {
	if !r.candidate.Found {
		return nil, nil
	}
	return []routing.Candidate{r.candidate}, nil
}
