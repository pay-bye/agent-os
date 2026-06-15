package instructions

import (
	"context"
	"errors"
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/kernel/routing"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

const maxIDs = 100

var (
	ErrSelectionEmpty = errors.New("instruction selection is empty")
	ErrSelectionLimit = errors.New("instruction selection exceeds limit")
	ErrDuplicateID    = errors.New("instruction selection has duplicate identity")
	ErrLimit          = errors.New("instruction limit is outside bounds")
)

type ItemsInput struct {
	ID        ID
	WorkItems []workitem.ID
}

type ItemsCommand struct {
	Record      Record
	WorkItems   []workitem.ID
	Events      []journal.EventID
	RouteEvents []journal.EventID
	Entries     []channel.EntryID
}

type WorkItemState struct {
	ID       workitem.ID
	Exists   bool
	Terminal bool
	Leased   bool
}

type TargetState struct {
	Channel  registry.ChannelKey
	Found    bool
	Excluded bool
}

type SourceState struct {
	Channel registry.ChannelKey
	Found   bool
}

type EntryState struct {
	ID     channel.EntryID
	Found  bool
	Leased bool
}

func itemsCommand(kind string, input ItemsInput, clock Clock) (ItemsCommand, error) {
	if err := validateID(input.ID); err != nil {
		return ItemsCommand{}, err
	}
	if err := validateWorkItemSelection(input.WorkItems); err != nil {
		return ItemsCommand{}, err
	}
	items := append([]workitem.ID(nil), input.WorkItems...)
	return ItemsCommand{
		Record:    record(kind, input, clock),
		WorkItems: items,
	}, nil
}

func workItemPrecondition(state WorkItemState) string {
	if !state.Exists {
		return "work_item_exists"
	}
	if state.Terminal {
		return "work_item_not_terminal"
	}
	if state.Leased {
		return "no_current_lease"
	}
	return ""
}

func targetPrecondition(state TargetState) string {
	if !state.Found {
		return "channel_exists"
	}
	if state.Excluded {
		return "target_channel_routable"
	}
	return ""
}

func sourcePrecondition(state SourceState) string {
	if !state.Found {
		return "channel_exists"
	}
	return ""
}

func entrySourcePrecondition(state EntryState) string {
	if !state.Found {
		return "entry_in_source"
	}
	return ""
}

func entryLeasePrecondition(state EntryState) string {
	if state.Leased {
		return "no_current_lease"
	}
	return ""
}

func entriesPrecondition(entries []EntryState) string {
	for _, entry := range entries {
		if failed := entrySourcePrecondition(entry); failed != "" {
			return failed
		}
		if failed := entryLeasePrecondition(entry); failed != "" {
			return failed
		}
	}
	return ""
}

func workItemsPrecondition(items []WorkItemState) (string, workitem.ID) {
	for _, item := range items {
		if failed := workItemPrecondition(item); failed != "" {
			return failed, item.ID
		}
	}
	return "", ""
}

func routePrecondition(
	ctx context.Context,
	items []RoutableWorkItem,
	routes routing.Facts,
) (string, workitem.ID) {
	for _, item := range items {
		if failed := routablePrecondition(ctx, item, routes); failed != "" {
			return failed, item.WorkItem.ID
		}
	}
	return "", ""
}

func routablePrecondition(ctx context.Context, item RoutableWorkItem, routes routing.Facts) string {
	if failed := workItemPrecondition(item.WorkItem); failed != "" {
		return failed
	}
	if item.Queued {
		return "need_unrouted"
	}
	if !item.NeedOpen {
		return "need_outstanding"
	}
	if _, err := routing.Select(ctx, routes, item.Need); err != nil {
		return "target_channel_routable"
	}
	return ""
}

func validateEntrySelection(entries []channel.EntryID) error {
	values := make([]string, 0, len(entries))
	for _, entry := range entries {
		values = append(values, entry.String())
	}
	return validateSelection(values)
}

func validateWorkItemSelection(items []workitem.ID) error {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, item.String())
	}
	return validateSelection(values)
}

func validateSelection(values []string) error {
	if len(values) == 0 {
		return ErrSelectionEmpty
	}
	if len(values) > maxIDs {
		return ErrSelectionLimit
	}
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" {
			return ErrSelectionEmpty
		}
		if seen[value] {
			return ErrDuplicateID
		}
		seen[value] = true
	}
	return nil
}
