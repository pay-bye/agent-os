package instructions

import (
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/journal/payloads"
	"github.com/pay-bye/agent-os/internal/workitem"
)

type DropFacts struct {
	WorkItems []WorkItemState
}

func Drop(input ItemsInput, clock Clock, ids IDs) (ItemsCommand, error) {
	command, err := itemsCommand("drop", input, clock)
	if err != nil {
		return ItemsCommand{}, err
	}
	command.Events = newEventIDs(ids, len(command.WorkItems))
	return command, nil
}

func ApplyDrop(command ItemsCommand, facts DropFacts) (Application, error) {
	scope := newScope(command.Record, command.Events[0], dropAudit(command.WorkItems))
	if failed, item := workItemsPrecondition(facts.WorkItems); failed != "" {
		return rejected(scope, journal.WorkItemCoordinate(item), failed)
	}
	outcomes, err := droppedOutcomes(command, scope.audit)
	if err != nil {
		return Application{}, err
	}
	return Application{
		Result:   result(command.Record, Applied, eventIDs(command.Events), workItemIDs(command.WorkItems), ""),
		Outcomes: outcomes,
		Effects:  Effects{Deletes: copyWorkItems(command.WorkItems)},
	}, nil
}

func dropAudit(items []workitem.ID) audit {
	return newAudit(
		map[string]any{"work_item_ids": workItemIDs(items)},
		"work_item_exists",
		"work_item_not_terminal",
		"no_current_lease",
	)
}

func droppedOutcomes(command ItemsCommand, audit audit) ([]Outcome, error) {
	outcomes := make([]Outcome, 0, len(command.WorkItems))
	for index, item := range command.WorkItems {
		scope := newScope(command.Record, command.Events[index], audit)
		outcome, err := outcome(
			scope,
			journal.WorkItemCoordinate(item),
			payloads.WorkItemDroppedKind,
			[]string{item.String()},
			"",
			Applied,
		)
		if err != nil {
			return nil, err
		}
		outcomes = append(outcomes, outcome)
	}
	return outcomes, nil
}
