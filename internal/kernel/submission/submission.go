package submission

import (
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
	"time"
)

type Input struct {
	ID            workitem.ID
	Kind          registry.ItemKindKey
	Payload       []byte
	DeclaredNeeds []workitem.DeclaredNeedInput
}

type Command struct {
	Submission  workitem.Submission
	SubmittedAt time.Time
	ItemEvent   journal.EventID
	NeedEvents  []journal.EventID
	RouteEvent  journal.EventID
	Entry       channel.EntryID
}

type IDs interface {
	Next() string
}

func New(input Input, now time.Time, ids IDs) (Command, error) {
	item, err := workitem.NewSubmission(workitem.SubmissionInput{
		ID:      input.ID,
		Kind:    input.Kind,
		Payload: input.Payload,
		Needs:   input.DeclaredNeeds,
	})
	if err != nil {
		return Command{}, err
	}
	return Command{
		Submission:  item,
		SubmittedAt: now,
		ItemEvent:   eventID(ids),
		NeedEvents:  eventIDs(ids, len(item.DeclaredNeeds())),
		RouteEvent:  eventID(ids),
		Entry:       channel.EntryID(ids.Next()),
	}, nil
}

func eventID(ids IDs) journal.EventID {
	return journal.EventID(ids.Next())
}

func eventIDs(ids IDs, count int) []journal.EventID {
	values := make([]journal.EventID, 0, count)
	for range count {
		values = append(values, eventID(ids))
	}
	return values
}
