package instructions

import (
	"context"
	"errors"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/kernel/pause"
	"github.com/pay-bye/agent-os/internal/registry"
)

type PauseInput struct {
	ID   ID
	Node registry.NodeKey
}

type PauseCommand struct {
	Record Record
	Node   registry.NodeKey
	Event  journal.EventID
}

func Pause(input PauseInput, clock Clock, ids IDs) (PauseCommand, error) {
	if err := validateID(input.ID); err != nil {
		return PauseCommand{}, err
	}
	return PauseCommand{
		Record: record("pause", input, clock),
		Node:   input.Node,
		Event:  eventID(ids),
	}, nil
}

func ApplyPause(ctx context.Context, command PauseCommand, facts pause.Facts) (Application, error) {
	scope := newScope(command.Record, command.Event, pauseAudit(command.Node))
	node, err := pause.Validate(ctx, facts, command.Node)
	if err != nil {
		return pauseRejected(scope, command.Node, err)
	}
	return applied(scope, journal.NodeCoordinate(node.Key()), []string{node.Key().String()}, Effects{
		Exclusions: []registry.NodeKey{node.Key()},
	})
}

func pauseAudit(node registry.NodeKey) audit {
	return newAudit(
		map[string]any{"node_key": node.String()},
		"node_installed",
		"node_has_alternate",
	)
}

func pauseRejected(scope applicationScope, node registry.NodeKey, err error) (Application, error) {
	if registry.IsNotFound(err) {
		return rejected(scope, journal.NodeCoordinate(node), "node_installed")
	}
	if !errors.Is(err, pause.ErrUnsafe) {
		return Application{}, err
	}
	return rejected(scope, journal.NodeCoordinate(node), "node_has_alternate")
}
