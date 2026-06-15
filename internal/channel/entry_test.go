package channel

import (
	"errors"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestNewEntryRejectsInvalidInput(t *testing.T) {
	for _, test := range []invalidEntryCase{
		{
			name: "empty identity",
			input: validEntryInput(func(input *EntryInput) {
				input.ID = ""
			}),
			want: ErrEmptyEntryID,
		},
		{
			name: "empty channel",
			input: validEntryInput(func(input *EntryInput) {
				input.Channel = ""
			}),
			want: ErrEmptyChannelKey,
		},
		{
			name: "empty work item",
			input: validEntryInput(func(input *EntryInput) {
				input.WorkItem = ""
			}),
			want: ErrEmptyWorkItemID,
		},
		{
			name: "zero enqueued time",
			input: validEntryInput(func(input *EntryInput) {
				input.EnqueuedAt = time.Time{}
			}),
			want: ErrMissingEnqueuedAt,
		},
		{
			name: "zero available time",
			input: validEntryInput(func(input *EntryInput) {
				input.AvailableAt = time.Time{}
			}),
			want: ErrMissingAvailableAt,
		},
	} {
		assertInvalidEntry(t, test)
	}
}

func TestNewEntryReportsFields(t *testing.T) {
	entry, err := NewEntry(validEntryInput())
	if err != nil {
		t.Fatal(err)
	}

	if entry.ID() != EntryID("x32") {
		t.Fatalf("entry identity = %q, want x32", entry.ID())
	}
	if entry.Channel() != registry.ChannelKey("x15") {
		t.Fatalf("channel = %q, want x15", entry.Channel())
	}
	if entry.WorkItem() != workitem.ID("x08") {
		t.Fatalf("work item = %q, want x08", entry.WorkItem())
	}
}

func validEntryInput(changes ...func(*EntryInput)) EntryInput {
	input := EntryInput{
		ID:          EntryID("x32"),
		Channel:     registry.ChannelKey("x15"),
		WorkItem:    workitem.ID("x08"),
		EnqueuedAt:  instant(0),
		AvailableAt: instant(0),
	}
	for _, change := range changes {
		change(&input)
	}
	return input
}

type invalidEntryCase struct {
	name  string
	input EntryInput
	want  error
}

func assertInvalidEntry(t *testing.T, test invalidEntryCase) {
	t.Helper()

	t.Run(test.name, func(t *testing.T) {
		_, err := NewEntry(test.input)

		if !errors.Is(err, test.want) {
			t.Fatalf("error = %v, want %v", err, test.want)
		}
	})
}
