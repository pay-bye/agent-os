package kernel

import (
	"context"
	"database/sql"
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/kernel/instructions"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func applyPauseInstruction(
	ctx context.Context,
	tx *sql.Tx,
	command instructions.PauseCommand,
) (instructions.Result, error) {
	application, err := instructions.ApplyPause(ctx, command, pauseFacts{tx: tx})
	if err != nil {
		return instructions.Result{}, err
	}
	return persistInstructionApplication(ctx, tx, application)
}

func applyReleaseExpiredLeaseInstruction(
	ctx context.Context,
	tx *sql.Tx,
	command instructions.LeaseCommand,
) (instructions.Result, error) {
	facts, err := leaseFacts(ctx, tx, command.Lease)
	if err != nil {
		return instructions.Result{}, err
	}
	application, err := instructions.ApplyReleaseExpiredLease(command, facts)
	if err != nil {
		return instructions.Result{}, err
	}
	return persistInstructionApplication(ctx, tx, application)
}

func applyForceReleaseLeaseInstruction(
	ctx context.Context,
	tx *sql.Tx,
	command instructions.LeaseCommand,
) (instructions.Result, error) {
	facts, err := leaseFacts(ctx, tx, command.Lease)
	if err != nil {
		return instructions.Result{}, err
	}
	application, err := instructions.ApplyForceReleaseLease(command, facts)
	if err != nil {
		return instructions.Result{}, err
	}
	return persistInstructionApplication(ctx, tx, application)
}

func applyMoveItemInstruction(
	ctx context.Context,
	tx *sql.Tx,
	command instructions.MoveItemCommand,
) (instructions.Result, error) {
	facts, err := moveItemFacts(ctx, tx, command)
	if err != nil {
		return instructions.Result{}, err
	}
	application, err := instructions.ApplyMoveItem(command, facts)
	if err != nil {
		return instructions.Result{}, err
	}
	return persistInstructionApplication(ctx, tx, application)
}

func applyMoveEntriesInstruction(
	ctx context.Context,
	tx *sql.Tx,
	command instructions.MoveEntriesCommand,
) (instructions.Result, error) {
	facts, err := moveEntriesFacts(ctx, tx, command)
	if err != nil {
		return instructions.Result{}, err
	}
	application, err := instructions.ApplyMoveEntries(command, facts)
	if err != nil {
		return instructions.Result{}, err
	}
	return persistInstructionApplication(ctx, tx, application)
}

func applyMoveAvailableInstruction(
	ctx context.Context,
	tx *sql.Tx,
	command instructions.MoveAvailableCommand,
) (instructions.Result, error) {
	facts, err := moveAvailableFacts(ctx, tx, command)
	if err != nil {
		return instructions.Result{}, err
	}
	application, err := instructions.ApplyMoveAvailable(command, facts)
	if err != nil {
		return instructions.Result{}, err
	}
	return persistInstructionApplication(ctx, tx, application)
}

func applyDropInstruction(
	ctx context.Context,
	tx *sql.Tx,
	command instructions.ItemsCommand,
) (instructions.Result, error) {
	facts, err := dropFacts(ctx, tx, command.WorkItems)
	if err != nil {
		return instructions.Result{}, err
	}
	application, err := instructions.ApplyDrop(command, facts)
	if err != nil {
		return instructions.Result{}, err
	}
	return persistInstructionApplication(ctx, tx, application)
}

func applyRouteOutstandingInstruction(
	ctx context.Context,
	tx *sql.Tx,
	command instructions.ItemsCommand,
) (instructions.Result, error) {
	facts, err := routeFacts(ctx, tx, command.WorkItems)
	if err != nil {
		return instructions.Result{}, err
	}
	application, err := instructions.ApplyRouteOutstanding(ctx, command, facts, routingFacts{tx: tx})
	if err != nil {
		return instructions.Result{}, err
	}
	return persistInstructionApplication(ctx, tx, application)
}

func persistInstructionApplication(
	ctx context.Context,
	tx *sql.Tx,
	application instructions.Application,
) (instructions.Result, error) {
	if err := applyInstructionEffects(ctx, tx, application.Effects); err != nil {
		return instructions.Result{}, err
	}
	if err := appendInstructionOutcomes(ctx, tx, application.Outcomes); err != nil {
		return instructions.Result{}, err
	}
	if err := applyInstructionRoutes(ctx, tx, application.Routes); err != nil {
		return instructions.Result{}, err
	}
	return application.Result, nil
}

func applyInstructionEffects(ctx context.Context, tx *sql.Tx, effects instructions.Effects) error {
	for _, node := range effects.Exclusions {
		if err := insertExclusion(ctx, tx, node); err != nil {
			return err
		}
	}
	for _, lease := range effects.Releases {
		if err := releaseLease(ctx, tx, lease); err != nil {
			return err
		}
	}
	for _, move := range effects.Moves {
		if err := moveEntries(ctx, tx, move.Entries, move.Target); err != nil {
			return err
		}
	}
	return deleteCurrentEntries(ctx, tx, effects.Deletes)
}

func appendInstructionOutcomes(ctx context.Context, tx *sql.Tx, outcomes []instructions.Outcome) error {
	for _, outcome := range outcomes {
		if err := appendInstructionOutcome(ctx, tx, outcome); err != nil {
			return err
		}
	}
	return nil
}

func appendInstructionOutcome(ctx context.Context, tx *sql.Tx, outcome instructions.Outcome) error {
	_, err := newJournal(tx).Append(ctx, journal.EventInput{
		ID:         outcome.Event,
		Coordinate: outcome.Coordinate,
		Kind:       outcome.Kind,
		AppendedAt: outcome.AppendedAt,
		Payload:    outcome.Payload,
	})
	return err
}

func applyInstructionRoutes(ctx context.Context, tx *sql.Tx, routes []instructions.RouteStep) error {
	for _, step := range routes {
		if err := appendInstructionOutcome(ctx, tx, step.Outcome); err != nil {
			return err
		}
		if err := routeSelectedInstruction(ctx, tx, step.Effect); err != nil {
			return err
		}
	}
	return nil
}

func routeSelectedInstruction(ctx context.Context, tx *sql.Tx, effect instructions.RouteEffect) error {
	command := routeCommand{
		WorkItem:  effect.WorkItem,
		Event:     effect.Event,
		Entry:     effect.Entry,
		CreatedAt: effect.CreatedAt,
	}
	if err := appendRouted(ctx, newJournal(tx), command, effect.Need, effect.Selection); err != nil {
		return err
	}
	_, err := newChannel(tx).Enqueue(ctx, channel.EntryInput{
		ID:          effect.Entry,
		Channel:     effect.Selection.Node.Channel(),
		WorkItem:    effect.WorkItem,
		EnqueuedAt:  effect.CreatedAt,
		AvailableAt: effect.CreatedAt,
	})
	return err
}

func leaseFacts(ctx context.Context, tx *sql.Tx, id channel.LeaseID) (instructions.LeaseFacts, error) {
	lease, found, err := lockLease(ctx, tx, id)
	return instructions.LeaseFacts{Lease: lease, Found: found}, err
}

func moveItemFacts(
	ctx context.Context,
	tx *sql.Tx,
	command instructions.MoveItemCommand,
) (instructions.MoveItemFacts, error) {
	work, err := workItemState(ctx, tx, command.WorkItem)
	if err != nil {
		return instructions.MoveItemFacts{}, err
	}
	target, err := targetState(ctx, tx, command.Target)
	if err != nil {
		return instructions.MoveItemFacts{}, err
	}
	entry, err := currentEntryState(ctx, tx, command.WorkItem, command.Source)
	if err != nil {
		return instructions.MoveItemFacts{}, err
	}
	return instructions.MoveItemFacts{WorkItem: work, Target: target, Entry: entry}, nil
}

func moveEntriesFacts(
	ctx context.Context,
	tx *sql.Tx,
	command instructions.MoveEntriesCommand,
) (instructions.MoveEntriesFacts, error) {
	source, err := sourceState(ctx, tx, command.Source)
	if err != nil {
		return instructions.MoveEntriesFacts{}, err
	}
	target, err := targetState(ctx, tx, command.Target)
	if err != nil {
		return instructions.MoveEntriesFacts{}, err
	}
	entries, err := entryStates(ctx, tx, command.Source, command.Entries)
	if err != nil {
		return instructions.MoveEntriesFacts{}, err
	}
	return instructions.MoveEntriesFacts{Source: source, Target: target, Entries: entries}, nil
}

func moveAvailableFacts(
	ctx context.Context,
	tx *sql.Tx,
	command instructions.MoveAvailableCommand,
) (instructions.MoveAvailableFacts, error) {
	source, err := sourceState(ctx, tx, command.Source)
	if err != nil {
		return instructions.MoveAvailableFacts{}, err
	}
	target, err := targetState(ctx, tx, command.Target)
	if err != nil {
		return instructions.MoveAvailableFacts{}, err
	}
	entries, err := oldestAvailableEntries(ctx, tx, command.Source, command.Limit)
	if err != nil {
		return instructions.MoveAvailableFacts{}, err
	}
	return instructions.MoveAvailableFacts{Source: source, Target: target, Entries: entries}, nil
}

func dropFacts(
	ctx context.Context,
	tx *sql.Tx,
	items []workitem.ID,
) (instructions.DropFacts, error) {
	states, err := workItemStates(ctx, tx, items)
	return instructions.DropFacts{WorkItems: states}, err
}

func routeFacts(
	ctx context.Context,
	tx *sql.Tx,
	items []workitem.ID,
) (instructions.RouteFacts, error) {
	facts := make([]instructions.RoutableWorkItem, 0, len(items))
	for _, item := range items {
		fact, err := routeItemFact(ctx, tx, item)
		if err != nil {
			return instructions.RouteFacts{}, err
		}
		facts = append(facts, fact)
	}
	return instructions.RouteFacts{WorkItems: facts}, nil
}

func routeItemFact(
	ctx context.Context,
	tx *sql.Tx,
	item workitem.ID,
) (instructions.RoutableWorkItem, error) {
	state, err := workItemState(ctx, tx, item)
	if err != nil {
		return instructions.RoutableWorkItem{}, err
	}
	queued, err := itemQueued(ctx, tx, item)
	if err != nil {
		return instructions.RoutableWorkItem{}, err
	}
	need, found, err := outstandingInstructionNeed(ctx, tx, item)
	if err != nil {
		return instructions.RoutableWorkItem{}, err
	}
	return instructions.RoutableWorkItem{WorkItem: state, Queued: queued, Need: need, NeedOpen: found}, nil
}

func workItemStates(
	ctx context.Context,
	tx *sql.Tx,
	items []workitem.ID,
) ([]instructions.WorkItemState, error) {
	states := make([]instructions.WorkItemState, 0, len(items))
	for _, item := range items {
		state, err := workItemState(ctx, tx, item)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, nil
}

func workItemState(ctx context.Context, tx *sql.Tx, item workitem.ID) (instructions.WorkItemState, error) {
	exists, err := workItemExists(ctx, tx, item)
	if err != nil {
		return instructions.WorkItemState{}, err
	}
	terminal, err := workItemTerminal(ctx, tx, item)
	if err != nil {
		return instructions.WorkItemState{}, err
	}
	leased, err := itemLeased(ctx, tx, item)
	if err != nil {
		return instructions.WorkItemState{}, err
	}
	return instructions.WorkItemState{ID: item, Exists: exists, Terminal: terminal, Leased: leased}, nil
}

func sourceState(ctx context.Context, tx *sql.Tx, source registry.ChannelKey) (instructions.SourceState, error) {
	_, found, err := channelNode(ctx, tx, source)
	if err != nil {
		return instructions.SourceState{}, err
	}
	return instructions.SourceState{Channel: source, Found: found}, nil
}

func targetState(ctx context.Context, tx *sql.Tx, target registry.ChannelKey) (instructions.TargetState, error) {
	node, found, err := channelNode(ctx, tx, target)
	if err != nil || !found {
		return instructions.TargetState{Channel: target, Found: found}, err
	}
	excluded, err := isExcluded(ctx, tx, node)
	if err != nil {
		return instructions.TargetState{}, err
	}
	return instructions.TargetState{Channel: target, Found: true, Excluded: excluded}, nil
}

func currentEntryState(
	ctx context.Context,
	tx *sql.Tx,
	item workitem.ID,
	source registry.ChannelKey,
) (instructions.EntryState, error) {
	entry, found, err := currentEntry(ctx, tx, item, source)
	if err != nil || !found {
		return instructions.EntryState{Found: found}, err
	}
	return lockedEntryState(ctx, tx, entry.id)
}

func entryStates(
	ctx context.Context,
	tx *sql.Tx,
	source registry.ChannelKey,
	entries []channel.EntryID,
) ([]instructions.EntryState, error) {
	states := make([]instructions.EntryState, 0, len(entries))
	for _, entry := range entries {
		state, err := entryState(ctx, tx, source, entry)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, nil
}

func entryState(
	ctx context.Context,
	tx *sql.Tx,
	source registry.ChannelKey,
	entry channel.EntryID,
) (instructions.EntryState, error) {
	row, found, err := entryInSource(ctx, tx, entry, source)
	if err != nil || !found {
		return instructions.EntryState{ID: entry, Found: found}, err
	}
	return lockedEntryState(ctx, tx, row.id)
}

func lockedEntryState(ctx context.Context, tx *sql.Tx, entry channel.EntryID) (instructions.EntryState, error) {
	leased, err := entryLeased(ctx, tx, entry)
	if err != nil {
		return instructions.EntryState{}, err
	}
	return instructions.EntryState{ID: entry, Found: true, Leased: leased}, nil
}

func releaseLease(ctx context.Context, tx *sql.Tx, lease channel.LeaseID) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM leases WHERE id = $1`, lease.String())
	return err
}
