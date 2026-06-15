package routing

import (
	"context"
	"errors"
	"fmt"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/journal/payloads"
	"github.com/pay-bye/agent-os/internal/registry"
)

var (
	ErrUnexpectedResolution = errors.New("need resolution has no outstanding declaration")
	ErrRouteTargetAbsent    = errors.New("route target is absent")
	ErrRouteTargetIncapable = errors.New("route target is incapable")
	ErrRouteTargetExcluded  = errors.New("route target is excluded")
)

type Need struct {
	Kind   registry.NeedKindKey
	Target registry.NodeKey
}

type Facts interface {
	AddressedTarget(context.Context, Need) (Candidate, error)
	DefaultCandidates(context.Context, Need) ([]Candidate, error)
}

type Candidate struct {
	Node      registry.Node
	RuleOrder int
	Excluded  bool
	Found     bool
}

type Selection struct {
	Node      registry.Node
	RuleOrder int
}

func Select(ctx context.Context, facts Facts, need Need) (Selection, error) {
	if need.Target.String() != "" {
		return selectAddressed(ctx, facts, need)
	}
	return selectDefault(ctx, facts, need)
}

func OutstandingNeed(events []journal.Event) (Need, bool, error) {
	needs := []Need{}
	for _, event := range events {
		switch event.Kind() {
		case payloads.NeedDeclaredKind:
			need, err := needFromEvent(event)
			if err != nil {
				return Need{}, false, err
			}
			needs = append(needs, need)
		case payloads.NeedAckedKind, payloads.NeedNackedKind:
			if len(needs) == 0 {
				return Need{}, false, ErrUnexpectedResolution
			}
			needs = needs[1:]
		}
	}
	if len(needs) == 0 {
		return Need{}, false, nil
	}
	return needs[0], true, nil
}

func selectAddressed(ctx context.Context, facts Facts, need Need) (Selection, error) {
	candidate, err := facts.AddressedTarget(ctx, need)
	if err != nil {
		return Selection{}, err
	}
	if !candidate.Found {
		return Selection{}, fmt.Errorf("%w: %s", ErrRouteTargetAbsent, need.Target.String())
	}
	if !candidate.Node.HasCapability(need.Kind) {
		return Selection{}, fmt.Errorf("%w: %s", ErrRouteTargetIncapable, need.Target.String())
	}
	if candidate.Excluded {
		return Selection{}, fmt.Errorf("%w: %s", ErrRouteTargetExcluded, need.Target.String())
	}
	return selectionFrom(candidate), nil
}

func selectDefault(ctx context.Context, facts Facts, need Need) (Selection, error) {
	candidates, err := facts.DefaultCandidates(ctx, need)
	if err != nil {
		return Selection{}, err
	}
	for _, candidate := range candidates {
		if !candidate.Excluded {
			return selectionFrom(candidate), nil
		}
	}
	return Selection{}, registry.ErrNoRoute
}

func selectionFrom(candidate Candidate) Selection {
	return Selection{
		Node:      candidate.Node,
		RuleOrder: candidate.RuleOrder,
	}
}

func needFromEvent(event journal.Event) (Need, error) {
	need, err := payloads.NeedFromEvent(event)
	if err != nil {
		return Need{}, err
	}
	return Need{
		Kind:   need.Kind,
		Target: need.Target,
	}, nil
}
