package instructions

import (
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

type MoveItemInput struct {
	ID       ID
	WorkItem workitem.ID
	Source   registry.ChannelKey
	Target   registry.ChannelKey
}

type MoveEntriesInput struct {
	ID      ID
	Source  registry.ChannelKey
	Target  registry.ChannelKey
	Entries []channel.EntryID
}

type MoveAvailableInput struct {
	ID     ID
	Source registry.ChannelKey
	Target registry.ChannelKey
	Limit  int
}

type MoveItemCommand struct {
	Record   Record
	WorkItem workitem.ID
	Source   registry.ChannelKey
	Target   registry.ChannelKey
	Event    journal.EventID
}

type MoveEntriesCommand struct {
	Record  Record
	Source  registry.ChannelKey
	Target  registry.ChannelKey
	Entries []channel.EntryID
	Event   journal.EventID
}

type MoveAvailableCommand struct {
	Record Record
	Source registry.ChannelKey
	Target registry.ChannelKey
	Limit  int
	Event  journal.EventID
}

type MoveItemFacts struct {
	WorkItem WorkItemState
	Target   TargetState
	Entry    EntryState
}

type MoveEntriesFacts struct {
	Source  SourceState
	Target  TargetState
	Entries []EntryState
}

type MoveAvailableFacts struct {
	Source  SourceState
	Target  TargetState
	Entries []channel.EntryID
}

func MoveItem(input MoveItemInput, clock Clock, ids IDs) (MoveItemCommand, error) {
	if err := validateID(input.ID); err != nil {
		return MoveItemCommand{}, err
	}
	return MoveItemCommand{
		Record:   record("move_item", input, clock),
		WorkItem: input.WorkItem,
		Source:   input.Source,
		Target:   input.Target,
		Event:    eventID(ids),
	}, nil
}

func MoveEntries(input MoveEntriesInput, clock Clock, ids IDs) (MoveEntriesCommand, error) {
	if err := validateID(input.ID); err != nil {
		return MoveEntriesCommand{}, err
	}
	if err := validateEntrySelection(input.Entries); err != nil {
		return MoveEntriesCommand{}, err
	}
	return MoveEntriesCommand{
		Record:  record("move_entries", input, clock),
		Source:  input.Source,
		Target:  input.Target,
		Entries: append([]channel.EntryID(nil), input.Entries...),
		Event:   eventID(ids),
	}, nil
}

func MoveAvailable(input MoveAvailableInput, clock Clock, ids IDs) (MoveAvailableCommand, error) {
	if err := validateID(input.ID); err != nil {
		return MoveAvailableCommand{}, err
	}
	if input.Limit <= 0 || input.Limit > maxIDs {
		return MoveAvailableCommand{}, ErrLimit
	}
	return MoveAvailableCommand{
		Record: record("move_available", input, clock),
		Source: input.Source,
		Target: input.Target,
		Limit:  input.Limit,
		Event:  eventID(ids),
	}, nil
}

func ApplyMoveItem(command MoveItemCommand, facts MoveItemFacts) (Application, error) {
	scope := newScope(command.Record, command.Event, moveItemAudit(command.WorkItem, command.Source, command.Target))
	if failed := workItemPrecondition(facts.WorkItem); failed != "" {
		return rejected(scope, journal.WorkItemCoordinate(command.WorkItem), failed)
	}
	if failed := targetPrecondition(facts.Target); failed != "" {
		return rejected(scope, journal.ChannelCoordinate(command.Target), failed)
	}
	if failed := entrySourcePrecondition(facts.Entry); failed != "" {
		return rejected(scope, journal.WorkItemCoordinate(command.WorkItem), failed)
	}
	if failed := entryLeasePrecondition(facts.Entry); failed != "" {
		return rejected(scope, journal.WorkItemCoordinate(command.WorkItem), failed)
	}
	return applied(scope, journal.WorkItemCoordinate(command.WorkItem), []string{command.WorkItem.String()}, Effects{
		Moves: []MoveEffect{{Entries: []channel.EntryID{facts.Entry.ID}, Target: command.Target}},
	})
}

func ApplyMoveEntries(command MoveEntriesCommand, facts MoveEntriesFacts) (Application, error) {
	scope := newScope(command.Record, command.Event, moveEntriesAudit(command.Source, command.Target, command.Entries))
	if failed := sourcePrecondition(facts.Source); failed != "" {
		return rejected(scope, journal.ChannelCoordinate(command.Source), failed)
	}
	if failed := targetPrecondition(facts.Target); failed != "" {
		return rejected(scope, journal.ChannelCoordinate(command.Target), failed)
	}
	if failed := entriesPrecondition(facts.Entries); failed != "" {
		return rejected(scope, journal.ChannelCoordinate(command.Source), failed)
	}
	return applied(scope, journal.ChannelCoordinate(command.Source), entryIDs(command.Entries), Effects{
		Moves: []MoveEffect{{Entries: command.Entries, Target: command.Target}},
	})
}

func ApplyMoveAvailable(command MoveAvailableCommand, facts MoveAvailableFacts) (Application, error) {
	scope := newScope(command.Record, command.Event, moveAvailableAudit(command.Source, command.Target, command.Limit))
	if failed := sourcePrecondition(facts.Source); failed != "" {
		return rejected(scope, journal.ChannelCoordinate(command.Source), failed)
	}
	if failed := targetPrecondition(facts.Target); failed != "" {
		return rejected(scope, journal.ChannelCoordinate(command.Target), failed)
	}
	return applied(scope, journal.ChannelCoordinate(command.Source), entryIDs(facts.Entries), Effects{
		Moves: []MoveEffect{{Entries: facts.Entries, Target: command.Target}},
	})
}

func moveItemAudit(item workitem.ID, source registry.ChannelKey, target registry.ChannelKey) audit {
	return newAudit(
		map[string]any{
			"work_item_id":       item.String(),
			"source_channel_key": source.String(),
			"target_channel_key": target.String(),
		},
		"work_item_exists",
		"work_item_not_terminal",
		"channel_exists",
		"target_channel_routable",
		"entry_exists",
		"entry_in_source",
		"no_current_lease",
	)
}

func moveEntriesAudit(source registry.ChannelKey, target registry.ChannelKey, entries []channel.EntryID) audit {
	return newAudit(
		map[string]any{
			"source_channel_key": source.String(),
			"target_channel_key": target.String(),
			"entry_ids":          entryIDs(entries),
		},
		"channel_exists",
		"target_channel_routable",
		"entry_exists",
		"entry_in_source",
		"no_current_lease",
	)
}

func moveAvailableAudit(source registry.ChannelKey, target registry.ChannelKey, limit int) audit {
	return newAudit(
		map[string]any{
			"source_channel_key": source.String(),
			"target_channel_key": target.String(),
			"limit":              limit,
		},
		"channel_exists",
		"target_channel_routable",
		"limit",
	)
}
