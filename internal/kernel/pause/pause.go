package pause

import (
	"context"
	"errors"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/registry"
	"time"
)

var ErrUnsafe = errors.New("pause would remove routability")

type Facts interface {
	Target(context.Context, registry.NodeKey) (registry.Node, error)
	Candidates(context.Context) ([]Candidate, error)
}

type Candidate struct {
	Node     registry.Node
	Excluded bool
}

type Input struct {
	Node registry.NodeKey
}

type Command struct {
	Node     registry.NodeKey
	Event    journal.EventID
	PausedAt time.Time
}

type IDs interface {
	Next() string
}

func New(input Input, now time.Time, ids IDs) Command {
	return Command{
		Node:     input.Node,
		Event:    journal.EventID(ids.Next()),
		PausedAt: now,
	}
}

func Validate(ctx context.Context, facts Facts, key registry.NodeKey) (registry.Node, error) {
	target, err := facts.Target(ctx, key)
	if err != nil {
		return registry.Node{}, err
	}
	candidates, err := facts.Candidates(ctx)
	if err != nil {
		return registry.Node{}, err
	}
	if !canPause(target, candidates) {
		return registry.Node{}, ErrUnsafe
	}
	return target, nil
}

func canPause(target registry.Node, candidates []Candidate) bool {
	for _, need := range target.Capabilities() {
		if !hasAlternate(target.Key(), need, candidates) {
			return false
		}
	}
	return true
}

func hasAlternate(
	target registry.NodeKey,
	need registry.NeedKindKey,
	candidates []Candidate,
) bool {
	for _, candidate := range candidates {
		if canServe(candidate, target, need) {
			return true
		}
	}
	return false
}

func canServe(
	candidate Candidate,
	target registry.NodeKey,
	need registry.NeedKindKey,
) bool {
	return !candidate.Excluded &&
		candidate.Node.Key() != target &&
		candidate.Node.HasCapability(need)
}
