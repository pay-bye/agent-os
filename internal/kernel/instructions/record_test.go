package instructions

import (
	"slices"
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestInstructionResultCopiesSlices(t *testing.T) {
	events := []string{"x80"}
	affected := []string{"x08"}
	record := Record{ID: "x70"}

	result := result(record, Applied, events, affected, "")
	events[0] = "mutated"
	affected[0] = "mutated"

	if !slices.Equal(result.EventIDs, []string{"x80"}) {
		t.Fatalf("events = %v, want x80", result.EventIDs)
	}
	if !slices.Equal(result.AffectedIDs, []string{"x08"}) {
		t.Fatalf("affected = %v, want x08", result.AffectedIDs)
	}
}

func TestInstructionIDProjectionsUseStringValues(t *testing.T) {
	events := eventIDs([]journal.EventID{"x80", "x81"})
	items := workItemIDs([]workitem.ID{"x08", "x09"})
	entries := entryIDs([]channel.EntryID{"x31", "x32"})

	if !slices.Equal(events, []string{"x80", "x81"}) {
		t.Fatalf("events = %v, want x80/x81", events)
	}
	if !slices.Equal(items, []string{"x08", "x09"}) {
		t.Fatalf("items = %v, want x08/x09", items)
	}
	if !slices.Equal(entries, []string{"x31", "x32"}) {
		t.Fatalf("entries = %v, want x31/x32", entries)
	}
}

func TestReplayEqualityIncludesResultFacts(t *testing.T) {
	first := Result{
		ID:                 "x70",
		Result:             PreconditionFailed,
		EventIDs:           []string{"x80"},
		AffectedIDs:        []string{"x08"},
		FailedPrecondition: "lease_expired",
	}
	second := Result{
		ID:                 "x70",
		Result:             PreconditionFailed,
		EventIDs:           []string{"x80"},
		AffectedIDs:        []string{"x08"},
		FailedPrecondition: "lease_expired",
	}

	if !first.SameReplay(second) {
		t.Fatal("matching replay was not equal")
	}
	second.AffectedIDs[0] = "x09"
	if first.SameReplay(second) {
		t.Fatal("different replay was equal")
	}
}
