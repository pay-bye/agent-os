package readmodel

import (
	"context"
	"errors"
	"time"
)

var ErrUnavailable = errors.New("read surface unavailable")

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

type JournalQuery struct {
	WorkItem         string
	Limit            int
	AfterAppendIndex int64
}

type NodeQuery struct {
	Limit    int
	After    string
	NeedKind string
}

type ChannelList struct {
	Channels []Channel `json:"channels"`
}

type Channel struct {
	Key                       string `json:"channel_key"`
	Node                      string `json:"node_key"`
	Depth                     int    `json:"depth"`
	Available                 int    `json:"available"`
	OldestAvailableAgeSeconds int    `json:"oldest_available_age_seconds"`
}

type ChannelItemList struct {
	Items []ChannelItem `json:"items"`
}

type ChannelItem struct {
	Entry       string    `json:"entry_id"`
	WorkItem    string    `json:"work_item_id"`
	Channel     string    `json:"channel_key"`
	Node        string    `json:"node_key"`
	EnqueuedAt  time.Time `json:"enqueued_at"`
	AvailableAt time.Time `json:"available_at"`
	AgeSeconds  int       `json:"age_seconds"`
	Lease       *Lease    `json:"lease,omitempty"`
}

type Lease struct {
	ID        string    `json:"lease_id"`
	GrantedAt time.Time `json:"granted_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ItemDetail struct {
	WorkItem    string        `json:"work_item_id"`
	Kind        string        `json:"item_kind"`
	SubmittedAt time.Time     `json:"submitted_at"`
	Entry       *ItemEntry    `json:"channel_entry,omitempty"`
	Lease       *ItemLease    `json:"lease,omitempty"`
	Need        *NeedSnapshot `json:"outstanding_need,omitempty"`
}

type ItemEntry struct {
	Entry       string    `json:"entry_id"`
	Channel     string    `json:"channel_key"`
	Node        string    `json:"node_key"`
	EnqueuedAt  time.Time `json:"enqueued_at"`
	AvailableAt time.Time `json:"available_at"`
	AgeSeconds  int       `json:"age_seconds"`
}

type ItemLease struct {
	ID        string    `json:"lease_id"`
	Channel   string    `json:"channel_key"`
	GrantedAt time.Time `json:"granted_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type NeedSnapshot struct {
	Event  string    `json:"event_id"`
	Kind   string    `json:"need_kind"`
	Target string    `json:"target_node,omitempty"`
	At     time.Time `json:"declared_at"`
}

type JournalEventList struct {
	Events []JournalEvent `json:"events"`
}

type JournalEvent struct {
	Event       string         `json:"event_id"`
	Kind        string         `json:"event_kind"`
	AppendedAt  time.Time      `json:"appended_at"`
	AppendIndex int64          `json:"append_index"`
	Metadata    map[string]any `json:"metadata"`
}

type NodeList struct {
	Nodes []Node `json:"nodes"`
}

type Node struct {
	Key       string   `json:"node_key"`
	Channel   string   `json:"channel_key"`
	NeedKinds []string `json:"need_kinds"`
	Routable  bool     `json:"routable"`
}

type ChannelSource interface {
	Channels(context.Context, time.Time, ChannelQuery) ([]Channel, error)
}

type ChannelItemSource interface {
	ChannelItems(context.Context, time.Time, ChannelItemQuery) ([]ChannelItem, error)
}

type ItemSource interface {
	Item(context.Context, time.Time, string) (ItemDetail, error)
}

type ItemJournalSource interface {
	ItemJournal(context.Context, JournalQuery) ([]JournalEvent, error)
}

type NodeSource interface {
	Nodes(context.Context, NodeQuery) ([]Node, error)
}
