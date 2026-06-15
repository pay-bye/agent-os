package instructions

import (
	"context"
	"slices"
	"testing"

	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/kernel/pause"
	"github.com/pay-bye/agent-os/internal/registry"
)

func TestPauseCommandBuildsStoreCommand(t *testing.T) {
	command, err := Pause(PauseInput{
		ID:   "x70",
		Node: "x17",
	}, fixedClock{now: instant(0)}, &sequenceIDs{values: []string{"x80"}})
	if err != nil {
		t.Fatal(err)
	}

	if command.Record.Kind != "pause" {
		t.Fatalf("kind = %q, want pause", command.Record.Kind)
	}
	if command.Event != journal.EventID("x80") {
		t.Fatalf("event = %q, want x80", command.Event)
	}
}

func TestPauseRejectsEmptyInstructionID(t *testing.T) {
	_, err := Pause(PauseInput{Node: "x17"}, fixedClock{now: instant(0)}, &sequenceIDs{})

	requireError(t, err, ErrEmptyID)
}

func TestApplyPauseBuildsExclusionPlan(t *testing.T) {
	command := PauseCommand{Record: recordAt("x70"), Node: "x17", Event: "x80"}
	target := node(t, "x17", "x15", "x91")
	alternate := node(t, "x18", "x68", "x91")
	facts := staticPauseFacts{
		target:     target,
		candidates: []pause.Candidate{{Node: target}, {Node: alternate}},
	}

	application, err := ApplyPause(context.Background(), command, facts)

	if err != nil {
		t.Fatal(err)
	}
	requireResult(t, application.Result, Applied, "")
	if !slices.Equal(application.Effects.Exclusions, []registry.NodeKey{"x17"}) {
		t.Fatalf("exclusions = %v, want x17", application.Effects.Exclusions)
	}
}

func TestApplyPauseRejectsMissingNode(t *testing.T) {
	command := PauseCommand{Record: recordAt("x70"), Node: "x17", Event: "x80"}

	application, err := ApplyPause(context.Background(), command, staticPauseFacts{missing: true})

	if err != nil {
		t.Fatal(err)
	}
	requireResult(t, application.Result, PreconditionFailed, "node_installed")
}

type staticPauseFacts struct {
	target     registry.Node
	candidates []pause.Candidate
	missing    bool
}

func (f staticPauseFacts) Target(context.Context, registry.NodeKey) (registry.Node, error) {
	if f.missing {
		return registry.Node{}, registry.NodeNotFound("x17")
	}
	return f.target, nil
}

func (f staticPauseFacts) Candidates(context.Context) ([]pause.Candidate, error) {
	return f.candidates, nil
}
