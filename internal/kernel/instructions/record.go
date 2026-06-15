package instructions

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"slices"
	"time"
)

const (
	Applied            ResultValue = "applied"
	PreconditionFailed ResultValue = "precondition_failed"
)

var (
	ErrEmptyID  = errors.New("instruction identity is empty")
	ErrConflict = errors.New("instruction conflicts with existing request")
)

type ID string

func (id ID) String() string {
	return string(id)
}

type ResultValue string

type Record struct {
	ID            ID
	Kind          string
	RequestDigest string
	RecordedAt    time.Time
}

type Result struct {
	ID                 ID
	Result             ResultValue
	EventIDs           []string
	AffectedIDs        []string
	FailedPrecondition string
}

func (r Result) SameReplay(other Result) bool {
	return r.ID == other.ID &&
		r.Result == other.Result &&
		r.FailedPrecondition == other.FailedPrecondition &&
		slices.Equal(r.EventIDs, other.EventIDs) &&
		slices.Equal(r.AffectedIDs, other.AffectedIDs)
}

type Clock interface {
	Now() time.Time
}

type IDs interface {
	Next() string
}

func record(kind string, request any, clock Clock) Record {
	return Record{
		ID:            instructionID(request),
		Kind:          kind,
		RequestDigest: requestDigest(kind, request),
		RecordedAt:    clock.Now(),
	}
}

func eventID(ids IDs) journal.EventID {
	return journal.EventID(ids.Next())
}

func newEventIDs(ids IDs, count int) []journal.EventID {
	values := make([]journal.EventID, 0, count)
	for range count {
		values = append(values, eventID(ids))
	}
	return values
}

func newEntryIDs(ids IDs, count int) []channel.EntryID {
	values := make([]channel.EntryID, 0, count)
	for range count {
		values = append(values, channel.EntryID(ids.Next()))
	}
	return values
}

func instructionID(request any) ID {
	switch value := request.(type) {
	case PauseInput:
		return value.ID
	case LeaseInput:
		return value.ID
	case MoveItemInput:
		return value.ID
	case MoveEntriesInput:
		return value.ID
	case MoveAvailableInput:
		return value.ID
	case ItemsInput:
		return value.ID
	default:
		return ""
	}
}

func requestDigest(kind string, request any) string {
	body, err := json.Marshal(struct {
		Kind    string `json:"kind"`
		Request any    `json:"request"`
	}{Kind: kind, Request: request})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func validateID(id ID) error {
	if id.String() == "" {
		return ErrEmptyID
	}
	return nil
}
