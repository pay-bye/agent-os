package pause

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/registry"
)

func TestNewCommandUsesClockAndIdentifier(t *testing.T) {
	command := New(Input{Node: "x17"}, instant(0), &sequenceIDs{values: []string{"x45"}})

	if command.Node != registry.NodeKey("x17") {
		t.Fatalf("node = %q, want x17", command.Node)
	}
	if command.Event != journal.EventID("x45") {
		t.Fatalf("event = %q, want x45", command.Event)
	}
	if command.PausedAt != instant(0) {
		t.Fatalf("paused at = %s, want %s", command.PausedAt, instant(0))
	}
}

func TestValidatePauseFailsClosedWhenCapabilityWouldBeStranded(t *testing.T) {
	facts := pauseFacts{
		target: node(t, "x17", "x15", "x12"),
		candidates: []Candidate{
			{Node: node(t, "x18", "x68", "x13")},
		},
	}

	_, err := Validate(context.Background(), facts, registry.NodeKey("x17"))

	if !errors.Is(err, ErrUnsafe) {
		t.Fatalf("error = %v, want unsafe pause", err)
	}
}

func TestValidatePauseAllowsWhenEveryCapabilityHasAvailableAlternate(t *testing.T) {
	facts := pauseFacts{
		target: node(t, "x17", "x15", "x12", "x13"),
		candidates: []Candidate{
			{Node: node(t, "x18", "x68", "x12")},
			{Node: node(t, "x19", "x69", "x13")},
		},
	}

	target, err := Validate(context.Background(), facts, registry.NodeKey("x17"))
	if err != nil {
		t.Fatal(err)
	}

	if target.Key() != registry.NodeKey("x17") {
		t.Fatalf("pause target = %q, want x17", target.Key())
	}
}

func TestValidatePauseIgnoresExcludedAlternates(t *testing.T) {
	facts := pauseFacts{
		target: node(t, "x17", "x15", "x12"),
		candidates: []Candidate{
			{Node: node(t, "x18", "x68", "x12"), Excluded: true},
		},
	}

	_, err := Validate(context.Background(), facts, registry.NodeKey("x17"))

	if !errors.Is(err, ErrUnsafe) {
		t.Fatalf("error = %v, want unsafe pause", err)
	}
}

type pauseFacts struct {
	target     registry.Node
	candidates []Candidate
}

func (f pauseFacts) Target(context.Context, registry.NodeKey) (registry.Node, error) {
	return f.target, nil
}

func (f pauseFacts) Candidates(context.Context) ([]Candidate, error) {
	return f.candidates, nil
}

func node(t *testing.T, key string, channel string, capabilities ...string) registry.Node {
	t.Helper()

	needs := make([]registry.NeedKindKey, 0, len(capabilities))
	for _, capability := range capabilities {
		needs = append(needs, registry.NeedKindKey(capability))
	}
	item, err := registry.NewNode(registry.NodeInput{
		Key:          registry.NodeKey(key),
		Description:  "test node",
		Channel:      registry.ChannelKey(channel),
		Capabilities: needs,
	})
	if err != nil {
		t.Fatal(err)
	}
	return item
}

type sequenceIDs struct {
	values []string
}

func (s *sequenceIDs) Next() string {
	value := s.values[0]
	s.values = s.values[1:]
	return value
}

func instant(offset int) time.Time {
	return time.Date(2026, 5, 18, 12, offset, 0, 0, time.UTC)
}
