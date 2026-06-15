package journal

import (
	"encoding/json"
	"errors"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
	"time"
)

const (
	WorkItem CoordinateKind = "work_item"
	Node     CoordinateKind = "node"
	Channel  CoordinateKind = "channel"
	Lease    CoordinateKind = "lease"
)

var (
	ErrEmptyEventID       = errors.New("journal event identity is empty")
	ErrEmptyCoordinate    = errors.New("journal coordinate is empty")
	ErrUnknownCoordinate  = errors.New("journal coordinate is unknown")
	ErrEmptyKind          = errors.New("journal event kind is empty")
	ErrMissingAppendTime  = errors.New("journal append time is missing")
	ErrEmptyPayload       = errors.New("journal payload is empty")
	ErrMalformedPayload   = errors.New("journal payload is malformed JSON")
	ErrMissingAppendIndex = errors.New("journal append index is missing")
)

type EventID string

func (id EventID) String() string {
	return string(id)
}

type CoordinateKind string

type Coordinate struct {
	kind CoordinateKind
	key  string
}

func NewCoordinate(kind CoordinateKind, key string) (Coordinate, error) {
	coordinate := Coordinate{kind: kind, key: key}
	if err := coordinate.Validate(); err != nil {
		return Coordinate{}, err
	}
	return coordinate, nil
}

func (c Coordinate) Kind() CoordinateKind {
	return c.kind
}

func (c Coordinate) Key() string {
	return c.key
}

func (c Coordinate) Validate() error {
	if c.kind == "" || c.key == "" {
		return ErrEmptyCoordinate
	}
	if c.kind != WorkItem && c.kind != Node && c.kind != Channel && c.kind != Lease {
		return ErrUnknownCoordinate
	}
	return nil
}

type EventInput struct {
	ID         EventID
	Coordinate Coordinate
	Kind       registry.JournalEventKindKey
	AppendedAt time.Time
	Payload    []byte
}

type Event struct {
	id          EventID
	coordinate  Coordinate
	kind        registry.JournalEventKindKey
	appendedAt  time.Time
	appendIndex int64
	payload     []byte
}

func NewEvent(input EventInput) (Event, error) {
	if err := validateInput(input); err != nil {
		return Event{}, err
	}
	return event(input, 0), nil
}

func (e Event) ID() EventID {
	return e.id
}

func (e Event) Coordinate() Coordinate {
	return e.coordinate
}

func (e Event) Kind() registry.JournalEventKindKey {
	return e.kind
}

func (e Event) AppendedAt() time.Time {
	return e.appendedAt
}

func (e Event) AppendIndex() int64 {
	return e.appendIndex
}

func (e Event) Payload() []byte {
	return copyBytes(e.payload)
}

func (e Event) WorkItem() workitem.ID {
	if e.coordinate.kind != WorkItem {
		return ""
	}
	return workitem.ID(e.coordinate.key)
}

func NewRecordedEvent(input EventInput, appendIndex int64) (Event, error) {
	if err := validateInput(input); err != nil {
		return Event{}, err
	}
	if appendIndex <= 0 {
		return Event{}, ErrMissingAppendIndex
	}
	return event(input, appendIndex), nil
}

func WorkItemCoordinate(id workitem.ID) Coordinate {
	return Coordinate{kind: WorkItem, key: id.String()}
}

func NodeCoordinate(key registry.NodeKey) Coordinate {
	return Coordinate{kind: Node, key: key.String()}
}

func ChannelCoordinate(key registry.ChannelKey) Coordinate {
	return Coordinate{kind: Channel, key: key.String()}
}

func LeaseCoordinate(id string) Coordinate {
	return Coordinate{kind: Lease, key: id}
}

func event(input EventInput, appendIndex int64) Event {
	return Event{
		id:          input.ID,
		coordinate:  input.Coordinate,
		kind:        input.Kind,
		appendedAt:  input.AppendedAt,
		appendIndex: appendIndex,
		payload:     copyBytes(input.Payload),
	}
}

func validateInput(input EventInput) error {
	if input.ID.String() == "" {
		return ErrEmptyEventID
	}
	if err := input.Coordinate.Validate(); err != nil {
		return err
	}
	if input.Kind.String() == "" {
		return ErrEmptyKind
	}
	if input.AppendedAt.IsZero() {
		return ErrMissingAppendTime
	}
	if len(input.Payload) == 0 {
		return ErrEmptyPayload
	}
	if !json.Valid(input.Payload) {
		return ErrMalformedPayload
	}
	return nil
}

func copyBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	return append([]byte(nil), value...)
}
