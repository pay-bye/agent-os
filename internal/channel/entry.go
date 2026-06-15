package channel

import (
	"errors"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
	"time"
)

var (
	ErrEmptyEntryID       = errors.New("channel entry identity is empty")
	ErrEmptyChannelKey    = errors.New("channel key is empty")
	ErrEmptyWorkItemID    = errors.New("channel work item identity is empty")
	ErrMissingEnqueuedAt  = errors.New("channel entry enqueued time is missing")
	ErrMissingAvailableAt = errors.New("channel entry available time is missing")
)

type EntryID string

func (id EntryID) String() string {
	return string(id)
}

type EntryInput struct {
	ID          EntryID
	Channel     registry.ChannelKey
	WorkItem    workitem.ID
	EnqueuedAt  time.Time
	AvailableAt time.Time
}

type Entry struct {
	id          EntryID
	channel     registry.ChannelKey
	workItem    workitem.ID
	enqueuedAt  time.Time
	availableAt time.Time
}

func NewEntry(input EntryInput) (Entry, error) {
	if err := validateEntryInput(input); err != nil {
		return Entry{}, err
	}
	return Entry{
		id:          input.ID,
		channel:     input.Channel,
		workItem:    input.WorkItem,
		enqueuedAt:  input.EnqueuedAt,
		availableAt: input.AvailableAt,
	}, nil
}

func (e Entry) ID() EntryID {
	return e.id
}

func (e Entry) Channel() registry.ChannelKey {
	return e.channel
}

func (e Entry) WorkItem() workitem.ID {
	return e.workItem
}

func (e Entry) EnqueuedAt() time.Time {
	return e.enqueuedAt
}

func (e Entry) AvailableAt() time.Time {
	return e.availableAt
}

func validateEntryInput(input EntryInput) error {
	if blank(input.ID.String()) {
		return ErrEmptyEntryID
	}
	if blank(input.Channel.String()) {
		return ErrEmptyChannelKey
	}
	if blank(input.WorkItem.String()) {
		return ErrEmptyWorkItemID
	}
	if input.EnqueuedAt.IsZero() {
		return ErrMissingEnqueuedAt
	}
	if input.AvailableAt.IsZero() {
		return ErrMissingAvailableAt
	}
	return nil
}
