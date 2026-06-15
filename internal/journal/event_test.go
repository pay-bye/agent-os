package journal

import (
	"errors"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestNewEventAcceptsWorkItemAndNodeCoordinates(t *testing.T) {
	for _, input := range []EventInput{
		validEventInput(func(input *EventInput) {
			input.Coordinate = WorkItemCoordinate(workitem.ID("x08"))
		}),
		validEventInput(func(input *EventInput) {
			input.Coordinate = NodeCoordinate(registry.NodeKey("x17"))
		}),
		validEventInput(func(input *EventInput) {
			input.Coordinate = ChannelCoordinate(registry.ChannelKey("x15"))
		}),
		validEventInput(func(input *EventInput) {
			input.Coordinate = LeaseCoordinate("x16")
		}),
	} {
		event, err := NewEvent(input)
		if err != nil {
			t.Fatal(err)
		}
		if event.Coordinate().Key() == "" {
			t.Fatal("coordinate key must be present")
		}
	}
}

func TestNewCoordinateValidatesKindAndKey(t *testing.T) {
	coordinate, err := NewCoordinate(Node, "x17")
	if err != nil {
		t.Fatal(err)
	}
	if coordinate.Kind() != Node || coordinate.Key() != "x17" {
		t.Fatalf("coordinate = %+v", coordinate)
	}
	for _, test := range []struct {
		name string
		kind CoordinateKind
		key  string
		want error
	}{
		{name: "empty kind", key: "x17", want: ErrEmptyCoordinate},
		{name: "empty key", kind: Node, want: ErrEmptyCoordinate},
		{name: "unknown kind", kind: "x99", key: "x17", want: ErrUnknownCoordinate},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewCoordinate(test.kind, test.key)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestCoordinatesDescribeChannelAndLease(t *testing.T) {
	channel := ChannelCoordinate(registry.ChannelKey("x15"))
	lease := LeaseCoordinate("x16")

	if channel.Kind() != Channel || channel.Key() != "x15" {
		t.Fatalf("channel coordinate = %+v", channel)
	}
	if lease.Kind() != Lease || lease.Key() != "x16" {
		t.Fatalf("lease coordinate = %+v", lease)
	}
}

func TestNewEventRejectsInvalidInput(t *testing.T) {
	for _, test := range []invalidEventCase{
		{
			name: "empty identity",
			input: validEventInput(func(input *EventInput) {
				input.ID = ""
			}),
			want: ErrEmptyEventID,
		},
		{
			name: "empty coordinate",
			input: validEventInput(func(input *EventInput) {
				input.Coordinate = Coordinate{}
			}),
			want: ErrEmptyCoordinate,
		},
		{
			name: "unknown coordinate",
			input: validEventInput(func(input *EventInput) {
				input.Coordinate = Coordinate{kind: "x99", key: "x08"}
			}),
			want: ErrUnknownCoordinate,
		},
		{
			name: "empty kind",
			input: validEventInput(func(input *EventInput) {
				input.Kind = ""
			}),
			want: ErrEmptyKind,
		},
		{
			name: "missing append time",
			input: validEventInput(func(input *EventInput) {
				input.AppendedAt = time.Time{}
			}),
			want: ErrMissingAppendTime,
		},
		{
			name: "empty payload",
			input: validEventInput(func(input *EventInput) {
				input.Payload = nil
			}),
			want: ErrEmptyPayload,
		},
		{
			name: "malformed payload",
			input: validEventInput(func(input *EventInput) {
				input.Payload = []byte(`{"broken"`)
			}),
			want: ErrMalformedPayload,
		},
	} {
		assertInvalidEvent(t, test)
	}
}

func TestNewEventPreservesPayload(t *testing.T) {
	input := validEventInput()
	event, err := NewEvent(input)
	if err != nil {
		t.Fatal(err)
	}

	input.Payload[0] = '['
	got := event.Payload()
	got[0] = '['

	if string(event.Payload()) != `{"event":"x48"}` {
		t.Fatalf("event payload must not expose mutable alias")
	}
	if event.WorkItem() != workitem.ID("x08") {
		t.Fatalf("work item = %q, want x08", event.WorkItem())
	}
	if event.Coordinate() != WorkItemCoordinate(workitem.ID("x08")) {
		t.Fatalf("coordinate = %+v, want work item x08", event.Coordinate())
	}
}

func TestNewRecordedEventRejectsInvalidAppendIndex(t *testing.T) {
	_, err := NewRecordedEvent(validEventInput(), 0)

	if !errors.Is(err, ErrMissingAppendIndex) {
		t.Fatalf("error = %v, want missing append index", err)
	}
}

func TestNewRecordedEventReportsAppendIndex(t *testing.T) {
	event, err := NewRecordedEvent(validEventInput(), 7)
	if err != nil {
		t.Fatal(err)
	}

	if event.AppendIndex() != 7 {
		t.Fatalf("append index = %d, want 7", event.AppendIndex())
	}
}

func TestNewRecordedEventReportsNodeCoordinateAccessors(t *testing.T) {
	input := validEventInput(func(input *EventInput) {
		input.Coordinate = NodeCoordinate(registry.NodeKey("x17"))
	})
	event, err := NewRecordedEvent(input, 7)
	if err != nil {
		t.Fatal(err)
	}

	if event.ID() != EventID("x20") {
		t.Fatalf("id = %q, want x20", event.ID())
	}
	if event.Kind() != registry.JournalEventKindKey("x20") {
		t.Fatalf("kind = %q, want x20", event.Kind())
	}
	if !event.AppendedAt().Equal(input.AppendedAt) {
		t.Fatalf("appended at = %s, want %s", event.AppendedAt(), input.AppendedAt)
	}
	if event.WorkItem() != "" {
		t.Fatalf("work item = %q, want empty for node coordinate", event.WorkItem())
	}
}

func validEventInput(changes ...func(*EventInput)) EventInput {
	input := EventInput{
		ID:         EventID("x20"),
		Coordinate: WorkItemCoordinate(workitem.ID("x08")),
		Kind:       registry.JournalEventKindKey("x20"),
		AppendedAt: time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC),
		Payload:    []byte(`{"event":"x48"}`),
	}
	for _, change := range changes {
		change(&input)
	}
	return input
}

type invalidEventCase struct {
	name  string
	input EventInput
	want  error
}

func assertInvalidEvent(t *testing.T, test invalidEventCase) {
	t.Helper()

	t.Run(test.name, func(t *testing.T) {
		_, err := NewEvent(test.input)

		if !errors.Is(err, test.want) {
			t.Fatalf("error = %v, want %v", err, test.want)
		}
	})
}
