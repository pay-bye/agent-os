package resolution

import (
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/workitem"
	"time"
)

type Input struct {
	Lease          channel.LeaseID
	Token          channel.Token
	DeclaredNeeds  []workitem.DeclaredNeedInput
	FailurePayload []byte
}

type Command struct {
	Lease          channel.LeaseID
	TokenDigest    channel.Digest
	ResolvedAt     time.Time
	Event          journal.EventID
	NeedEvents     []journal.EventID
	RouteEvent     journal.EventID
	Entry          channel.EntryID
	DeclaredNeeds  []workitem.DeclaredNeed
	FailurePayload []byte
}

type IDs interface {
	Next() string
}

func New(input Input, now time.Time, ids IDs) (Command, error) {
	needs, err := workitem.NewDeclaredNeeds(input.DeclaredNeeds)
	if err != nil {
		return Command{}, err
	}
	digest, err := channel.DigestFor(input.Token)
	if err != nil {
		return Command{}, err
	}
	return Command{
		Lease:          input.Lease,
		TokenDigest:    digest,
		ResolvedAt:     now,
		Event:          eventID(ids),
		NeedEvents:     eventIDs(ids, len(needs)),
		RouteEvent:     eventID(ids),
		Entry:          channel.EntryID(ids.Next()),
		DeclaredNeeds:  needs,
		FailurePayload: copyBytes(input.FailurePayload),
	}, nil
}

func copyBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	return append([]byte(nil), value...)
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
