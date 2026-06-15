package operations

import (
	"context"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
	"time"
)

type Operations interface {
	Response(context.Context) OperationsReport
	Channels(context.Context, ChannelQuery) (any, error)
	ChannelItems(context.Context, ChannelItemQuery) (any, error)
	Item(context.Context, string) (any, error)
	ItemJournal(context.Context, ItemJournalQuery) (any, error)
	Nodes(context.Context, NodeQuery) (any, error)
}

type OperationsReport interface {
	Available() bool
}

type ChannelQuery struct {
	Limit            int
	After            string
	OlderThanSeconds int
}

type ChannelItemQuery struct {
	Channel          string
	Limit            int
	OlderThanSeconds int
	Lease            string
}

type ItemJournalQuery struct {
	WorkItem         string
	Limit            int
	AfterAppendIndex int64
}

type NodeQuery struct {
	Limit    int
	After    string
	NeedKind string
}

type unavailableOperations struct{}

func (unavailableOperations) Response(context.Context) OperationsReport {
	return unavailableReport{
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Result:        "partial",
		WindowSeconds: 300,
		Views:         map[string]any{},
		Unavailable:   unavailableGroups(),
	}
}

func (unavailableOperations) Channels(context.Context, ChannelQuery) (any, error) {
	return nil, codec.ErrUnavailable
}

func (unavailableOperations) ChannelItems(context.Context, ChannelItemQuery) (any, error) {
	return nil, codec.ErrUnavailable
}

func (unavailableOperations) Item(context.Context, string) (any, error) {
	return nil, codec.ErrUnavailable
}

func (unavailableOperations) ItemJournal(context.Context, ItemJournalQuery) (any, error) {
	return nil, codec.ErrUnavailable
}

func (unavailableOperations) Nodes(context.Context, NodeQuery) (any, error) {
	return nil, codec.ErrUnavailable
}

type unavailableReport struct {
	GeneratedAt   string         `json:"generated_at"`
	Result        string         `json:"result"`
	WindowSeconds int            `json:"window_seconds"`
	Views         map[string]any `json:"views"`
	Unavailable   []string       `json:"unavailable"`
}

func (unavailableReport) Available() bool {
	return false
}

func Unavailable() Operations {
	return unavailableOperations{}
}

func unavailableGroups() []string {
	return []string{
		"queue",
		"leases",
		"journal",
		"commands",
		"routing",
		"build",
		"compatibility",
	}
}
