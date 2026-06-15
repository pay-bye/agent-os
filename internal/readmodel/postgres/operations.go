package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/pay-bye/agent-os/internal/journal/payloads"
	"github.com/pay-bye/agent-os/internal/readmodel"
	"strings"
	"time"
)

const pressureQuery = `
SELECT
  (
    SELECT COUNT(*)
    FROM channel_entries
  ) AS depth,
  (
    SELECT COUNT(*)
    FROM channel_entries e
    WHERE e.available_at <= $1
      AND NOT EXISTS (
        SELECT 1
        FROM leases l
        WHERE l.channel_entry_id = e.id
          AND l.expires_at > $1
      )
  ) AS available,
  (
    SELECT COUNT(*)
    FROM leases
    WHERE expires_at > $1
  ) AS held,
  (
    SELECT COUNT(*)
    FROM leases
    WHERE expires_at <= $1
  ) AS expired,
  COALESCE((
    SELECT FLOOR(EXTRACT(EPOCH FROM ($1 - MIN(e.available_at))))::INT
    FROM channel_entries e
    WHERE e.available_at <= $1
      AND NOT EXISTS (
        SELECT 1
        FROM leases l
        WHERE l.channel_entry_id = e.id
          AND l.expires_at > $1
      )
  ), 0) AS oldest_available_age_seconds`

const journalQuery = `
SELECT COUNT(*)
FROM journal_events
WHERE appended_at > $1
  AND appended_at <= $2`

const (
	LeaseViewAll     = "all"
	LeaseViewHeld    = "held"
	LeaseViewExpired = "expired"
	LeaseViewNone    = "none"
)

type Pressure struct {
	Depth                     int
	Available                 int
	Held                      int
	Expired                   int
	OldestAvailableAgeSeconds int
}

type JournalWindow struct {
	Appends       int
	WindowSeconds int
}

type ChannelQuery = readmodel.ChannelQuery
type ChannelSummary = readmodel.Channel
type ChannelItemQuery = readmodel.ChannelItemQuery
type ChannelItem = readmodel.ChannelItem
type Lease = readmodel.Lease
type ItemDetail = readmodel.ItemDetail
type ItemEntry = readmodel.ItemEntry
type ItemLease = readmodel.ItemLease
type NeedSnapshot = readmodel.NeedSnapshot
type JournalQuery = readmodel.JournalQuery
type JournalEvent = readmodel.JournalEvent
type NodeQuery = readmodel.NodeQuery
type Node = readmodel.Node

type leaseFields struct {
	ID        sql.NullString
	GrantedAt sql.NullTime
	ExpiresAt sql.NullTime
}

type itemEntry struct {
	ID             sql.NullString
	WorkItem       sql.NullString
	Channel        sql.NullString
	Node           sql.NullString
	EnqueuedAt     sql.NullTime
	AvailableAt    sql.NullTime
	AgeSeconds     sql.NullInt64
	LeaseID        sql.NullString
	LeaseChannel   sql.NullString
	LeaseGrantedAt sql.NullTime
	LeaseExpiresAt sql.NullTime
}

type Operations struct {
	query queryFunc
	rows  rowsQueryFunc
}

func NewOperations(db rowReader) *Operations {
	return &Operations{
		query: func(ctx context.Context, query string, args ...any) rowScanner {
			return db.QueryRowContext(ctx, query, args...)
		},
		rows: func(ctx context.Context, query string, args ...any) (rowsScanner, error) {
			return db.QueryContext(ctx, query, args...)
		},
	}
}

func (o *Operations) Pressure(ctx context.Context, now time.Time) (Pressure, error) {
	var pressure Pressure
	err := o.query(ctx, pressureQuery,
		now,
	).Scan(
		&pressure.Depth,
		&pressure.Available,
		&pressure.Held,
		&pressure.Expired,
		&pressure.OldestAvailableAgeSeconds,
	)
	return pressure, err
}

func (o *Operations) Journal(ctx context.Context, now time.Time, window time.Duration) (JournalWindow, error) {
	since := now.Add(-window)
	item := JournalWindow{WindowSeconds: int(window.Seconds())}
	err := o.query(ctx, journalQuery,
		since,
		now,
	).Scan(&item.Appends)
	return item, err
}

func (o *Operations) Channels(ctx context.Context, now time.Time, query ChannelQuery) ([]ChannelSummary, error) {
	rows, err := o.rows(ctx, `
WITH available_entries AS (
  SELECT e.*
  FROM channel_entries e
  WHERE e.available_at <= $1
    AND NOT EXISTS (
      SELECT 1
      FROM leases l
      WHERE l.channel_entry_id = e.id
        AND l.expires_at > $1
    )
)
SELECT
  c.key,
  c.node_key,
  COUNT(e.id)::INT AS depth,
  COUNT(a.id)::INT AS available,
  COALESCE(FLOOR(EXTRACT(EPOCH FROM ($1 - MIN(a.available_at))))::INT, 0) AS oldest_available_age_seconds
FROM channels c
LEFT JOIN channel_entries e ON e.channel_key = c.key
LEFT JOIN available_entries a ON a.id = e.id
WHERE c.key > $2
GROUP BY c.key, c.node_key
HAVING $3 = 0 OR COALESCE(FLOOR(EXTRACT(EPOCH FROM ($1 - MIN(a.available_at))))::INT, 0) >= $3
ORDER BY c.key
LIMIT $4`,
		now,
		query.After,
		query.OlderThanSeconds,
		query.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChannels(rows)
}

func (o *Operations) ChannelItems(ctx context.Context, now time.Time, query ChannelItemQuery) ([]ChannelItem, error) {
	rows, err := o.rows(ctx, `
SELECT
  e.id,
  e.work_item_id,
  e.channel_key,
  c.node_key,
  e.enqueued_at,
  e.available_at,
  FLOOR(EXTRACT(EPOCH FROM ($1 - e.available_at)))::INT AS age_seconds,
  l.id,
  l.granted_at,
  l.expires_at
FROM channel_entries e
JOIN channels c ON c.key = e.channel_key
LEFT JOIN leases l ON l.channel_entry_id = e.id
WHERE e.channel_key = $2
  AND e.available_at <= $1
  AND ($3 = 0 OR FLOOR(EXTRACT(EPOCH FROM ($1 - e.available_at)))::INT >= $3)
  AND (
    $4 = 'all'
    OR ($4 = 'held' AND l.expires_at > $1)
    OR ($4 = 'expired' AND l.expires_at <= $1)
    OR ($4 = 'none' AND l.id IS NULL)
  )
ORDER BY e.available_at, e.enqueued_at, e.id
LIMIT $5`,
		now,
		query.Channel,
		query.OlderThanSeconds,
		query.Lease,
		query.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanItems(rows)
}

func (o *Operations) Item(ctx context.Context, now time.Time, id string) (ItemDetail, error) {
	item, err := o.item(ctx, now, id)
	if err != nil {
		return ItemDetail{}, err
	}
	need, err := o.outstandingNeed(ctx, id)
	if err != nil {
		return ItemDetail{}, err
	}
	item.Need = need
	return item, nil
}

func (o *Operations) item(ctx context.Context, now time.Time, id string) (ItemDetail, error) {
	var item ItemDetail
	var entry itemEntry
	err := o.query(ctx, `
SELECT
  w.id,
  w.item_kind_key,
  w.submitted_at,
  e.id,
  e.work_item_id,
  e.channel_key,
  c.node_key,
  e.enqueued_at,
  e.available_at,
  FLOOR(EXTRACT(EPOCH FROM ($2 - e.available_at)))::INT,
  l.id,
  l.channel_key,
  l.granted_at,
  l.expires_at
FROM work_items w
LEFT JOIN channel_entries e ON e.work_item_id = w.id
LEFT JOIN channels c ON c.key = e.channel_key
LEFT JOIN leases l ON l.channel_entry_id = e.id
WHERE w.id = $1
ORDER BY e.available_at, e.enqueued_at, e.id
LIMIT 1`,
		id,
		now,
	).Scan(
		&item.WorkItem,
		&item.Kind,
		&item.SubmittedAt,
		&entry.ID,
		&entry.WorkItem,
		&entry.Channel,
		&entry.Node,
		&entry.EnqueuedAt,
		&entry.AvailableAt,
		&entry.AgeSeconds,
		&entry.LeaseID,
		&entry.LeaseChannel,
		&entry.LeaseGrantedAt,
		&entry.LeaseExpiresAt,
	)
	if err != nil {
		return ItemDetail{}, err
	}
	addItemEntry(&item, entry)
	return item, nil
}

func (o *Operations) outstandingNeed(ctx context.Context, id string) (*NeedSnapshot, error) {
	var need NeedSnapshot
	var target sql.NullString
	err := o.query(ctx, `
SELECT
  d.id,
  d.payload->>'need_kind',
  d.payload->>'target_node',
  d.appended_at
FROM journal_events d
WHERE d.coordinate_kind = 'work_item'
  AND d.coordinate_key = $1
  AND d.event_kind_key = $2
  AND NOT EXISTS (
    SELECT 1
    FROM journal_events r
    WHERE r.coordinate_kind = 'work_item'
      AND r.coordinate_key = d.coordinate_key
      AND r.event_kind_key IN ($3, $4)
      AND r.append_index > d.append_index
  )
ORDER BY d.append_index
LIMIT 1`,
		id,
		payloads.NeedDeclaredKind.String(),
		payloads.NeedAckedKind.String(),
		payloads.NeedNackedKind.String(),
	).Scan(&need.Event, &need.Kind, &target, &need.At)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	need.Target = target.String
	return &need, nil
}

func (o *Operations) ItemJournal(ctx context.Context, query JournalQuery) ([]JournalEvent, error) {
	rows, err := o.rows(ctx, `
SELECT
  id,
  event_kind_key,
  appended_at,
  append_index,
  jsonb_strip_nulls(jsonb_build_object(
    'work_item_id', payload->'work_item_id',
    'item_kind', payload->'item_kind',
    'need_kind', payload->'need_kind',
    'target_node', payload->'target_node',
    'channel_key', payload->'channel_key',
    'node_key', payload->'node_key',
    'lease_id', payload->'lease_id',
    'entry_id', payload->'entry_id',
    'routing_rule_order', payload->'routing_rule_order',
    'result', payload->'result'
  ))
FROM journal_events
WHERE coordinate_kind = 'work_item'
  AND coordinate_key = $1
  AND append_index > $2
ORDER BY append_index
LIMIT $3`,
		query.WorkItem,
		query.AfterAppendIndex,
		query.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJournalEvents(rows)
}

func (o *Operations) Nodes(ctx context.Context, query NodeQuery) ([]Node, error) {
	rows, err := o.rows(ctx, `
SELECT
  n.key,
  c.key,
  string_agg(nc.need_kind_key, ',' ORDER BY nc.need_kind_key),
  NOT EXISTS (
    SELECT 1
    FROM routing_exclusions e
    WHERE e.node_key = n.key
  ) AS routable
FROM nodes n
JOIN channels c ON c.node_key = n.key
JOIN node_capabilities nc ON nc.node_key = n.key
WHERE n.key > $1
  AND (
    $2 = ''
    OR EXISTS (
      SELECT 1
      FROM node_capabilities filtered
      WHERE filtered.node_key = n.key
        AND filtered.need_kind_key = $2
    )
  )
GROUP BY n.key, c.key
ORDER BY n.key
LIMIT $3`,
		query.After,
		query.NeedKind,
		query.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

func scanNodes(rows rowsScanner) ([]Node, error) {
	var nodes []Node
	for rows.Next() {
		node, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

func scanNode(row rowScanner) (Node, error) {
	var node Node
	var kinds string
	if err := row.Scan(&node.Key, &node.Channel, &kinds, &node.Routable); err != nil {
		return Node{}, err
	}
	node.NeedKinds = strings.Split(kinds, ",")
	return node, nil
}

func scanChannels(rows rowsScanner) ([]ChannelSummary, error) {
	var channels []ChannelSummary
	for rows.Next() {
		channel, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	return channels, rows.Err()
}

func scanChannel(row rowScanner) (ChannelSummary, error) {
	var channel ChannelSummary
	err := row.Scan(
		&channel.Key,
		&channel.Node,
		&channel.Depth,
		&channel.Available,
		&channel.OldestAvailableAgeSeconds,
	)
	return channel, err
}

func scanItems(rows rowsScanner) ([]ChannelItem, error) {
	var items []ChannelItem
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanItem(row rowScanner) (ChannelItem, error) {
	var item ChannelItem
	var lease leaseFields
	err := row.Scan(
		&item.Entry,
		&item.WorkItem,
		&item.Channel,
		&item.Node,
		&item.EnqueuedAt,
		&item.AvailableAt,
		&item.AgeSeconds,
		&lease.ID,
		&lease.GrantedAt,
		&lease.ExpiresAt,
	)
	if err != nil {
		return ChannelItem{}, err
	}
	item.Lease = leaseFromFields(lease)
	return item, nil
}

func leaseFromFields(fields leaseFields) *Lease {
	if !fields.ID.Valid {
		return nil
	}
	return &Lease{
		ID:        fields.ID.String,
		GrantedAt: fields.GrantedAt.Time,
		ExpiresAt: fields.ExpiresAt.Time,
	}
}

func addItemEntry(item *ItemDetail, entry itemEntry) {
	if !entry.ID.Valid {
		return
	}
	item.Entry = &ItemEntry{
		Entry:       entry.ID.String,
		Channel:     entry.Channel.String,
		Node:        entry.Node.String,
		EnqueuedAt:  entry.EnqueuedAt.Time,
		AvailableAt: entry.AvailableAt.Time,
		AgeSeconds:  int(entry.AgeSeconds.Int64),
	}
	if !entry.LeaseID.Valid {
		return
	}
	item.Lease = &ItemLease{
		ID:        entry.LeaseID.String,
		Channel:   entry.LeaseChannel.String,
		GrantedAt: entry.LeaseGrantedAt.Time,
		ExpiresAt: entry.LeaseExpiresAt.Time,
	}
}

func scanJournalEvents(rows rowsScanner) ([]JournalEvent, error) {
	var events []JournalEvent
	for rows.Next() {
		event, err := scanJournalEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func scanJournalEvent(row rowScanner) (JournalEvent, error) {
	var event JournalEvent
	var metadata []byte
	err := row.Scan(
		&event.Event,
		&event.Kind,
		&event.AppendedAt,
		&event.AppendIndex,
		&metadata,
	)
	if err != nil {
		return JournalEvent{}, err
	}
	if err := json.Unmarshal(metadata, &event.Metadata); err != nil {
		return JournalEvent{}, err
	}
	return event, nil
}
