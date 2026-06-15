package instructions

import (
	"context"
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/journal/payloads"
	"github.com/pay-bye/agent-os/internal/kernel/routing"
	"github.com/pay-bye/agent-os/internal/workitem"
	"time"
)

type RouteStep struct {
	Outcome Outcome
	Effect  RouteEffect
}

type RouteEffect struct {
	WorkItem  workitem.ID
	Event     journal.EventID
	Entry     channel.EntryID
	CreatedAt time.Time
	Need      routing.Need
	Selection routing.Selection
}

type RouteFacts struct {
	WorkItems []RoutableWorkItem
}

type RoutableWorkItem struct {
	WorkItem WorkItemState
	Queued   bool
	Need     routing.Need
	NeedOpen bool
}

func RouteOutstanding(input ItemsInput, clock Clock, ids IDs) (ItemsCommand, error) {
	command, err := itemsCommand("route_outstanding", input, clock)
	if err != nil {
		return ItemsCommand{}, err
	}
	command.Events = newEventIDs(ids, len(command.WorkItems))
	command.RouteEvents = newEventIDs(ids, len(command.WorkItems))
	command.Entries = newEntryIDs(ids, len(command.WorkItems))
	return command, nil
}

func ApplyRouteOutstanding(
	ctx context.Context,
	command ItemsCommand,
	facts RouteFacts,
	routes routing.Facts,
) (Application, error) {
	scope := newScope(command.Record, command.Events[0], routeAudit(command.WorkItems, command.Entries))
	if failed, item := routePrecondition(ctx, facts.WorkItems, routes); failed != "" {
		return rejected(scope, journal.WorkItemCoordinate(item), failed)
	}
	return routeApplication(ctx, command, facts.WorkItems, routes, scope.audit)
}

func routeAudit(items []workitem.ID, entries []channel.EntryID) audit {
	return newAudit(
		map[string]any{
			"work_item_ids": workItemIDs(items),
			"entry_ids":     entryIDs(entries),
		},
		"work_item_exists",
		"work_item_not_terminal",
		"no_current_lease",
		"need_outstanding",
		"need_unrouted",
		"target_channel_routable",
	)
}

func routeApplication(
	ctx context.Context,
	command ItemsCommand,
	facts []RoutableWorkItem,
	routes routing.Facts,
	audit audit,
) (Application, error) {
	steps := make([]RouteStep, 0, len(command.WorkItems))
	events := make([]journal.EventID, 0, len(command.WorkItems)*2)
	for index := range command.WorkItems {
		step, err := routeStep(ctx, command, facts[index], routes, audit, index)
		if err != nil {
			return Application{}, err
		}
		steps = append(steps, step)
		events = append(events, command.Events[index], command.RouteEvents[index])
	}
	return Application{
		Result: result(command.Record, Applied, eventIDs(events), workItemIDs(command.WorkItems), ""),
		Routes: steps,
	}, nil
}

func routeStep(
	ctx context.Context,
	command ItemsCommand,
	fact RoutableWorkItem,
	routes routing.Facts,
	audit audit,
	index int,
) (RouteStep, error) {
	scope := newScope(command.Record, command.Events[index], audit)
	outcome, err := outcome(
		scope,
		journal.WorkItemCoordinate(command.WorkItems[index]),
		payloads.InstructionAppliedKind,
		[]string{command.WorkItems[index].String()},
		"",
		Applied,
	)
	if err != nil {
		return RouteStep{}, err
	}
	selection, err := routing.Select(ctx, routes, fact.Need)
	if err != nil {
		return RouteStep{}, err
	}
	return RouteStep{
		Outcome: outcome,
		Effect: RouteEffect{
			WorkItem:  command.WorkItems[index],
			Event:     command.RouteEvents[index],
			Entry:     command.Entries[index],
			CreatedAt: command.Record.RecordedAt,
			Need:      fact.Need,
			Selection: selection,
		},
	}, nil
}
