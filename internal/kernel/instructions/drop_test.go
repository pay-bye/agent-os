package instructions

import (
	"testing"

	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/journal/payloads"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestDropCommandBuildsStoreCommand(t *testing.T) {
	command, err := Drop(ItemsInput{
		ID:        "x70",
		WorkItems: []workitem.ID{"x08", "x09"},
	}, fixedClock{now: instant(0)}, &sequenceIDs{values: []string{"x80", "x81"}})
	if err != nil {
		t.Fatal(err)
	}

	if command.Record.Kind != "drop" {
		t.Fatalf("kind = %q, want drop", command.Record.Kind)
	}
	if command.Events[0] != journal.EventID("x80") || command.Events[1] != journal.EventID("x81") {
		t.Fatalf("events = %v, want x80/x81", command.Events)
	}
}

func TestApplyDropBuildsDropPlan(t *testing.T) {
	command := ItemsCommand{Record: recordAt("x70"), WorkItems: []workitem.ID{"x08", "x09"}, Events: []journal.EventID{"x80", "x81"}}
	facts := DropFacts{WorkItems: []WorkItemState{{ID: "x08", Exists: true}, {ID: "x09", Exists: true}}}

	application, err := ApplyDrop(command, facts)

	if err != nil {
		t.Fatal(err)
	}
	requireResult(t, application.Result, Applied, "")
	if len(application.Outcomes) != 2 || len(application.Effects.Deletes) != 2 {
		t.Fatalf("drop plan = %+v", application)
	}
	requireOutcome(t, application.Outcomes[0], payloads.WorkItemDroppedKind)
}
