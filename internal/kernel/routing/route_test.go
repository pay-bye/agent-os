package routing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/journal/payloads"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestOutstandingNeedUsesEarliestUnresolvedDeclaration(t *testing.T) {
	events := []journal.Event{
		recordedEvent(t, "x22", payloads.NeedDeclaredKind, `{"need_kind":"x12","payload":{"value":"one"}}`, 1),
		recordedEvent(t, "x23", payloads.NeedDeclaredKind, `{"need_kind":"x13","payload":{"value":"two"}}`, 2),
		recordedEvent(t, "x24", payloads.NeedAckedKind, `{"lease_id":"x16","work_item_id":"x08"}`, 3),
	}

	need, ok, err := OutstandingNeed(events)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("expected outstanding need")
	}
	if need.Kind != registry.NeedKindKey("x13") {
		t.Fatalf("need kind = %q, want x13", need.Kind)
	}
}

func TestOutstandingNeedPreservesDeclaredTarget(t *testing.T) {
	events := []journal.Event{
		recordedEvent(t, "x22", payloads.NeedDeclaredKind, `{"need_kind":"x12","target_node":"x17","payload":{"value":"x48"}}`, 1),
	}

	need, ok, err := OutstandingNeed(events)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("expected outstanding need")
	}
	if need.Target != registry.NodeKey("x17") {
		t.Fatalf("need target = %q, want x17", need.Target)
	}
}

func TestOutstandingNeedReturnsFalseWhenAllDeclarationsResolved(t *testing.T) {
	events := []journal.Event{
		recordedEvent(t, "x22", payloads.NeedDeclaredKind, `{"need_kind":"x12","payload":{"value":"one"}}`, 1),
		recordedEvent(t, "x23", payloads.NeedNackedKind, `{"lease_id":"x16","work_item_id":"x08"}`, 2),
	}

	_, ok, err := OutstandingNeed(events)
	if err != nil {
		t.Fatal(err)
	}

	if ok {
		t.Fatal("expected no outstanding need")
	}
}

func TestOutstandingNeedRejectsResolutionWithoutDeclaration(t *testing.T) {
	events := []journal.Event{
		recordedEvent(t, "x22", payloads.NeedAckedKind, `{"lease_id":"x16","work_item_id":"x08"}`, 1),
	}

	_, _, err := OutstandingNeed(events)

	if err != ErrUnexpectedResolution {
		t.Fatalf("error = %v, want unexpected resolution", err)
	}
}

func TestOutstandingNeedRejectsMalformedDeclaration(t *testing.T) {
	events := []journal.Event{
		recordedEvent(t, "x22", payloads.NeedDeclaredKind, `{"payload":{"value":"one"}}`, 1),
	}

	_, _, err := OutstandingNeed(events)

	if err != payloads.ErrMalformedNeed {
		t.Fatalf("error = %v, want malformed need event", err)
	}
}

func TestSelectAddressedTargetFailsClosedWhenAbsent(t *testing.T) {
	facts := routeFacts{
		addressed: Candidate{Found: false},
	}

	_, err := Select(context.Background(), facts, Need{
		Kind:   registry.NeedKindKey("x12"),
		Target: registry.NodeKey("x99"),
	})

	if !errors.Is(err, ErrRouteTargetAbsent) {
		t.Fatalf("error = %v, want absent route target", err)
	}
}

func TestSelectAddressedTargetFailsClosedWhenIncapable(t *testing.T) {
	facts := routeFacts{
		addressed: Candidate{
			Found: true,
			Node:  node(t, "x17", "x15", "x13"),
		},
	}

	_, err := Select(context.Background(), facts, Need{
		Kind:   registry.NeedKindKey("x12"),
		Target: registry.NodeKey("x17"),
	})

	if !errors.Is(err, ErrRouteTargetIncapable) {
		t.Fatalf("error = %v, want incapable route target", err)
	}
}

func TestSelectAddressedTargetFailsClosedWhenExcluded(t *testing.T) {
	facts := routeFacts{
		addressed: Candidate{
			Found:    true,
			Node:     node(t, "x17", "x15", "x12"),
			Excluded: true,
		},
	}

	_, err := Select(context.Background(), facts, Need{
		Kind:   registry.NeedKindKey("x12"),
		Target: registry.NodeKey("x17"),
	})

	if !errors.Is(err, ErrRouteTargetExcluded) {
		t.Fatalf("error = %v, want excluded route target", err)
	}
}

func TestSelectDefaultSkipsExcludedCandidates(t *testing.T) {
	facts := routeFacts{
		defaults: []Candidate{
			{Found: true, Node: node(t, "x17", "x15", "x12"), RuleOrder: 1, Excluded: true},
			{Found: true, Node: node(t, "x18", "x68", "x12"), RuleOrder: 2},
		},
	}

	selection, err := Select(context.Background(), facts, Need{Kind: registry.NeedKindKey("x12")})
	if err != nil {
		t.Fatal(err)
	}

	if selection.Node.Key() != registry.NodeKey("x18") {
		t.Fatalf("route target = %q, want x18", selection.Node.Key())
	}
	if selection.RuleOrder != 2 {
		t.Fatalf("route order = %d, want 2", selection.RuleOrder)
	}
}

func TestSelectDefaultFailsClosedWhenAllCandidatesAreExcluded(t *testing.T) {
	facts := routeFacts{
		defaults: []Candidate{
			{Found: true, Node: node(t, "x17", "x15", "x12"), RuleOrder: 1, Excluded: true},
		},
	}

	_, err := Select(context.Background(), facts, Need{Kind: registry.NeedKindKey("x12")})

	if !errors.Is(err, registry.ErrNoRoute) {
		t.Fatalf("error = %v, want no route", err)
	}
}

func recordedEvent(t *testing.T, id string, kind registry.JournalEventKindKey, payload string, index int64) journal.Event {
	t.Helper()

	event, err := journal.NewRecordedEvent(journal.EventInput{
		ID:         journal.EventID(id),
		Coordinate: journal.WorkItemCoordinate(workitem.ID("x08")),
		Kind:       kind,
		AppendedAt: time.Date(2026, 5, 18, 12, int(index), 0, 0, time.UTC),
		Payload:    []byte(payload),
	}, index)
	if err != nil {
		t.Fatal(err)
	}
	return event
}

type routeFacts struct {
	addressed Candidate
	defaults  []Candidate
}

func (f routeFacts) AddressedTarget(context.Context, Need) (Candidate, error) {
	return f.addressed, nil
}

func (f routeFacts) DefaultCandidates(context.Context, Need) ([]Candidate, error) {
	return f.defaults, nil
}

func node(t *testing.T, key string, channel string, capabilities ...string) registry.Node {
	t.Helper()

	needKinds := make([]registry.NeedKindKey, 0, len(capabilities))
	for _, capability := range capabilities {
		needKinds = append(needKinds, registry.NeedKindKey(capability))
	}
	node, err := registry.NewNode(registry.NodeInput{
		Key:          registry.NodeKey(key),
		Description:  "test node",
		Channel:      registry.ChannelKey(channel),
		Capabilities: needKinds,
	})
	if err != nil {
		t.Fatal(err)
	}
	return node
}
