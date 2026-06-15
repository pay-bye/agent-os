package kernel

import (
	"context"
	"database/sql"
	coordination "github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/journal/payloads"
	"github.com/pay-bye/agent-os/internal/kernel/routing"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
	"time"
)

type routeResult struct {
	Routed  bool
	Channel registry.ChannelKey
}

type routeCommand struct {
	WorkItem  workitem.ID
	Event     journal.EventID
	Entry     coordination.EntryID
	CreatedAt time.Time
}

type routingFacts struct {
	tx *sql.Tx
}

func (f routingFacts) AddressedTarget(ctx context.Context, need routing.Need) (routing.Candidate, error) {
	node, err := newRegistry(f.tx).FindNode(ctx, need.Target)
	if err != nil {
		if registry.IsNotFound(err) {
			return routing.Candidate{Found: false}, nil
		}
		return routing.Candidate{}, err
	}
	excluded, err := isExcluded(ctx, f.tx, node.Key())
	if err != nil {
		return routing.Candidate{}, err
	}
	return routing.Candidate{Found: true, Node: node, Excluded: excluded}, nil
}

func (f routingFacts) DefaultCandidates(ctx context.Context, need routing.Need) ([]routing.Candidate, error) {
	rules, err := newRegistry(f.tx).FindRoutingRules(ctx, need.Kind)
	if err != nil {
		return nil, err
	}
	candidates := make([]routing.Candidate, 0, len(rules))
	for _, rule := range rules {
		candidate, err := f.candidate(ctx, rule)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

func (f routingFacts) candidate(ctx context.Context, rule registry.RoutingRule) (routing.Candidate, error) {
	node, err := newRegistry(f.tx).FindNode(ctx, rule.Node())
	if err != nil {
		return routing.Candidate{}, err
	}
	excluded, err := isExcluded(ctx, f.tx, rule.Node())
	if err != nil {
		return routing.Candidate{}, err
	}
	return routing.Candidate{
		Found:     true,
		Node:      node,
		RuleOrder: rule.Order(),
		Excluded:  excluded,
	}, nil
}

func routeOutstanding(ctx context.Context, tx *sql.Tx, command routeCommand) (routeResult, error) {
	events := newJournal(tx)
	replay, err := events.Replay(ctx, command.WorkItem)
	if err != nil {
		return routeResult{}, err
	}
	need, ok, err := routing.OutstandingNeed(replay)
	if err != nil || !ok {
		return routeResult{}, err
	}
	return route(ctx, tx, command, need)
}

func outstandingInstructionNeed(
	ctx context.Context,
	tx *sql.Tx,
	item workitem.ID,
) (routing.Need, bool, error) {
	events, err := newJournal(tx).Replay(ctx, item)
	if err != nil {
		return routing.Need{}, false, err
	}
	return routing.OutstandingNeed(events)
}

func route(ctx context.Context, tx *sql.Tx, command routeCommand, need routing.Need) (routeResult, error) {
	selection, err := routing.Select(ctx, routingFacts{tx: tx}, need)
	if err != nil {
		return routeResult{}, err
	}
	if err := appendRouted(ctx, newJournal(tx), command, need, selection); err != nil {
		return routeResult{}, err
	}
	_, err = newChannel(tx).Enqueue(ctx, coordination.EntryInput{
		ID:          command.Entry,
		Channel:     selection.Node.Channel(),
		WorkItem:    command.WorkItem,
		EnqueuedAt:  command.CreatedAt,
		AvailableAt: command.CreatedAt,
	})
	if err != nil {
		return routeResult{}, err
	}
	return routeResult{Routed: true, Channel: selection.Node.Channel()}, nil
}

func isExcluded(ctx context.Context, tx *sql.Tx, key registry.NodeKey) (bool, error) {
	var found int
	err := tx.QueryRowContext(ctx, `
SELECT 1
FROM routing_exclusions
WHERE node_key = $1`, key.String()).Scan(&found)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
}

func appendRouted(
	ctx context.Context,
	events *journalStore,
	command routeCommand,
	need routing.Need,
	selection routing.Selection,
) error {
	payload, err := payloads.ItemRouted(payloads.RoutedItem{
		NeedKind:  need.Kind,
		Node:      selection.Node,
		RuleOrder: selection.RuleOrder,
	})
	if err != nil {
		return err
	}
	_, err = events.Append(ctx, journal.EventInput{
		ID:         command.Event,
		Coordinate: journal.WorkItemCoordinate(command.WorkItem),
		Kind:       payloads.ItemRoutedKind,
		AppendedAt: command.CreatedAt,
		Payload:    payload,
	})
	return err
}
