package instructions

import (
	"slices"
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestMoveItemCommandBuildsStoreCommand(t *testing.T) {
	command, err := MoveItem(MoveItemInput{
		ID:       "x70",
		WorkItem: "x08",
		Source:   "x15",
		Target:   "x68",
	}, fixedClock{now: instant(0)}, &sequenceIDs{values: []string{"x80"}})
	if err != nil {
		t.Fatal(err)
	}

	if command.Record.Kind != "move_item" {
		t.Fatalf("kind = %q, want move_item", command.Record.Kind)
	}
	if command.WorkItem != workitem.ID("x08") || command.Source != registry.ChannelKey("x15") {
		t.Fatalf("move item command = %+v", command)
	}
}

func TestMoveEntriesCommandBuildsStoreCommand(t *testing.T) {
	command, err := MoveEntries(MoveEntriesInput{
		ID:      "x70",
		Source:  "x15",
		Target:  "x68",
		Entries: []channel.EntryID{"x31", "x32"},
	}, fixedClock{now: instant(0)}, &sequenceIDs{values: []string{"x80"}})
	if err != nil {
		t.Fatal(err)
	}

	if command.Record.Kind != "move_entries" {
		t.Fatalf("kind = %q, want move_entries", command.Record.Kind)
	}
	if len(command.Entries) != 2 || command.Entries[1] != channel.EntryID("x32") {
		t.Fatalf("entries = %v, want x31/x32", command.Entries)
	}
}

func TestMoveAvailableCommandBuildsStoreCommand(t *testing.T) {
	command, err := MoveAvailable(MoveAvailableInput{
		ID:     "x70",
		Source: "x15",
		Target: "x68",
		Limit:  2,
	}, fixedClock{now: instant(0)}, &sequenceIDs{values: []string{"x80"}})
	if err != nil {
		t.Fatal(err)
	}

	if command.Record.Kind != "move_available" {
		t.Fatalf("kind = %q, want move_available", command.Record.Kind)
	}
	if command.Limit != 2 {
		t.Fatalf("limit = %d, want 2", command.Limit)
	}
}

func TestMoveEntriesRejectsDuplicateEntries(t *testing.T) {
	_, err := MoveEntries(MoveEntriesInput{
		ID:      "x70",
		Entries: []channel.EntryID{"x31", "x31"},
	}, fixedClock{now: instant(0)}, &sequenceIDs{})

	requireError(t, err, ErrDuplicateID)
}

func TestMoveAvailableRejectsInvalidLimit(t *testing.T) {
	_, err := MoveAvailable(MoveAvailableInput{ID: "x70"}, fixedClock{now: instant(0)}, &sequenceIDs{})

	requireError(t, err, ErrLimit)
}

func TestApplyMoveItemBuildsMovePlan(t *testing.T) {
	command := MoveItemCommand{Record: recordAt("x70"), WorkItem: "x08", Source: "x15", Target: "x68", Event: "x80"}
	facts := MoveItemFacts{
		WorkItem: WorkItemState{ID: "x08", Exists: true},
		Target:   TargetState{Channel: "x68", Found: true},
		Entry:    EntryState{ID: "x31", Found: true},
	}

	application, err := ApplyMoveItem(command, facts)

	if err != nil {
		t.Fatal(err)
	}
	requireResult(t, application.Result, Applied, "")
	requireMove(t, application.Effects.Moves[0], "x68", "x31")
}

func TestApplyMoveItemRejectsMissingWorkItem(t *testing.T) {
	command := MoveItemCommand{Record: recordAt("x70"), WorkItem: "x08", Source: "x15", Target: "x68", Event: "x80"}

	application, err := ApplyMoveItem(command, MoveItemFacts{})

	if err != nil {
		t.Fatal(err)
	}
	requireResult(t, application.Result, PreconditionFailed, "work_item_exists")
}

func TestApplyMoveEntriesRejectsLeasedEntry(t *testing.T) {
	command := MoveEntriesCommand{Record: recordAt("x70"), Source: "x15", Target: "x68", Entries: []channel.EntryID{"x31"}, Event: "x80"}
	facts := MoveEntriesFacts{
		Source:  SourceState{Channel: "x15", Found: true},
		Target:  TargetState{Channel: "x68", Found: true},
		Entries: []EntryState{{ID: "x31", Found: true, Leased: true}},
	}

	application, err := ApplyMoveEntries(command, facts)

	if err != nil {
		t.Fatal(err)
	}
	requireResult(t, application.Result, PreconditionFailed, "no_current_lease")
}

func TestApplyMoveAvailableBuildsMovePlan(t *testing.T) {
	command := MoveAvailableCommand{Record: recordAt("x70"), Source: "x15", Target: "x68", Limit: 2, Event: "x80"}
	facts := MoveAvailableFacts{
		Source:  SourceState{Channel: "x15", Found: true},
		Target:  TargetState{Channel: "x68", Found: true},
		Entries: []channel.EntryID{"x31", "x32"},
	}

	application, err := ApplyMoveAvailable(command, facts)

	if err != nil {
		t.Fatal(err)
	}
	requireResult(t, application.Result, Applied, "")
	requireMove(t, application.Effects.Moves[0], "x68", "x31", "x32")
}

func requireMove(t *testing.T, move MoveEffect, target registry.ChannelKey, entries ...channel.EntryID) {
	t.Helper()

	if move.Target != target || !slices.Equal(move.Entries, entries) {
		t.Fatalf("move = %+v, want target %s entries %v", move, target, entries)
	}
}
