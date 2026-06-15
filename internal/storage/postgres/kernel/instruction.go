package kernel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/journal/payloads"
	"github.com/pay-bye/agent-os/internal/kernel/instructions"
	"github.com/pay-bye/agent-os/internal/registry"
	channelstore "github.com/pay-bye/agent-os/internal/storage/postgres/channel"
	eventstore "github.com/pay-bye/agent-os/internal/storage/postgres/journal"
	"github.com/pay-bye/agent-os/internal/workitem"
	"strings"
	"time"
)

type instructionRecordRow struct {
	id       string
	kind     string
	digest   string
	eventIDs textArray
}

type entryRow struct {
	id       channel.EntryID
	workItem workitem.ID
}

type textArray []string

func (a *textArray) Scan(value any) error {
	switch content := value.(type) {
	case nil:
		*a = []string{}
		return nil
	case string:
		return a.scan(content)
	case []byte:
		return a.scan(string(content))
	default:
		return fmt.Errorf("unsupported text array %T", value)
	}
}

func (a *textArray) scan(value string) error {
	if value == "{}" {
		*a = []string{}
		return nil
	}
	if !strings.HasPrefix(value, "{") || !strings.HasSuffix(value, "}") {
		return fmt.Errorf("malformed text array")
	}
	parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(value, "{"), "}"), ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		values = append(values, strings.Trim(part, `"`))
	}
	*a = values
	return nil
}

func reserveInstruction(
	ctx context.Context,
	tx *sql.Tx,
	record instructions.Record,
) (instructions.Result, bool, error) {
	rows, err := insertInstructionReservation(ctx, tx, record)
	if err != nil {
		return instructions.Result{}, false, err
	}
	if rows == 1 {
		return instructions.Result{}, false, nil
	}
	return replayInstruction(ctx, tx, record)
}

func insertInstructionReservation(
	ctx context.Context,
	tx *sql.Tx,
	record instructions.Record,
) (int64, error) {
	result, err := tx.ExecContext(ctx, `
INSERT INTO instruction_records (instruction_id, kind, request_digest, recorded_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (instruction_id) DO NOTHING`,
		record.ID.String(),
		record.Kind,
		record.RequestDigest,
		record.RecordedAt,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func replayInstruction(
	ctx context.Context,
	tx *sql.Tx,
	record instructions.Record,
) (instructions.Result, bool, error) {
	row, err := findInstruction(ctx, tx, record.ID)
	if err != nil {
		return instructions.Result{}, false, err
	}
	if row.kind != record.Kind || row.digest != record.RequestDigest {
		return instructions.Result{}, false, instructions.ErrConflict
	}
	events, err := findInstructionEvents(ctx, tx, row.eventIDs)
	if err != nil {
		return instructions.Result{}, false, err
	}
	result, err := resultFromEvents(row, events)
	return result, true, err
}

func findInstruction(ctx context.Context, tx *sql.Tx, id instructions.ID) (instructionRecordRow, error) {
	var row instructionRecordRow
	err := tx.QueryRowContext(ctx, `
SELECT instruction_id, kind, request_digest, event_ids
FROM instruction_records
WHERE instruction_id = $1`, id.String()).Scan(
		&row.id,
		&row.kind,
		&row.digest,
		&row.eventIDs,
	)
	return row, err
}

func findInstructionEvents(
	ctx context.Context,
	tx *sql.Tx,
	ids []string,
) ([]journal.Event, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("instruction record has no journal events")
	}
	rows, err := tx.QueryContext(ctx, `
SELECT id, coordinate_kind, coordinate_key, event_kind_key, appended_at, append_index, payload
FROM journal_events
WHERE id = ANY($1)
ORDER BY array_position($1::text[], id)`, []string(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]journal.Event, 0, len(ids))
	for rows.Next() {
		event, err := eventstore.ScanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(events) != len(ids) {
		return nil, fmt.Errorf("instruction journal event reference is missing")
	}
	return events, nil
}

func resultFromEvents(row instructionRecordRow, events []journal.Event) (instructions.Result, error) {
	return instructions.ReplayResult(instructions.ReplayRecord{
		ID:       instructions.ID(row.id),
		EventIDs: append([]string(nil), row.eventIDs...),
	}, events)
}

func finishInstruction(ctx context.Context, tx *sql.Tx, result instructions.Result) error {
	_, err := tx.ExecContext(ctx, `
UPDATE instruction_records
SET event_ids = $2,
    recorded_at = $3
WHERE instruction_id = $1`,
		result.ID.String(),
		result.EventIDs,
		time.Now().UTC(),
	)
	return err
}

func lockLease(ctx context.Context, tx *sql.Tx, id channel.LeaseID) (channel.Lease, bool, error) {
	lease, err := channelstore.LeaseFromRow(tx.QueryRowContext(ctx, `
SELECT id, channel_entry_id, work_item_id, channel_key, granted_at, expires_at
FROM leases
WHERE id = $1
FOR UPDATE`, id.String()))
	if errors.Is(err, channel.ErrEmpty) {
		return channel.Lease{}, false, nil
	}
	return lease, err == nil, err
}

func workItemExists(ctx context.Context, tx *sql.Tx, item workitem.ID) (bool, error) {
	return existsRow(ctx, tx, `SELECT 1 FROM work_items WHERE id = $1`, item.String())
}

func workItemTerminal(ctx context.Context, tx *sql.Tx, item workitem.ID) (bool, error) {
	return existsRow(ctx, tx, `
SELECT 1
FROM journal_events
WHERE coordinate_kind = 'work_item'
  AND coordinate_key = $1
  AND event_kind_key = $2`, item.String(), payloads.WorkItemDroppedKind.String())
}

func itemLeased(ctx context.Context, tx *sql.Tx, item workitem.ID) (bool, error) {
	return existsRow(ctx, tx, `SELECT 1 FROM leases WHERE work_item_id = $1`, item.String())
}

func itemQueued(ctx context.Context, tx *sql.Tx, item workitem.ID) (bool, error) {
	return existsRow(ctx, tx, `SELECT 1 FROM channel_entries WHERE work_item_id = $1`, item.String())
}

func entryLeased(ctx context.Context, tx *sql.Tx, entry channel.EntryID) (bool, error) {
	return existsRow(ctx, tx, `SELECT 1 FROM leases WHERE channel_entry_id = $1`, entry.String())
}

func existsRow(ctx context.Context, tx *sql.Tx, query string, args ...any) (bool, error) {
	var found int
	err := tx.QueryRowContext(ctx, query, args...).Scan(&found)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func channelNode(ctx context.Context, tx *sql.Tx, channel registry.ChannelKey) (registry.NodeKey, bool, error) {
	var node string
	err := tx.QueryRowContext(ctx, `SELECT node_key FROM channels WHERE key = $1`, channel.String()).Scan(&node)
	if err == nil {
		return registry.NodeKey(node), true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return "", false, err
}

func currentEntry(
	ctx context.Context,
	tx *sql.Tx,
	item workitem.ID,
	source registry.ChannelKey,
) (entryRow, bool, error) {
	var row entryRow
	var id string
	err := tx.QueryRowContext(ctx, `
SELECT id
FROM channel_entries
WHERE work_item_id = $1
  AND channel_key = $2
FOR UPDATE`, item.String(), source.String()).Scan(&id)
	if err == nil {
		row.id = channel.EntryID(id)
		row.workItem = item
		return row, true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return entryRow{}, false, nil
	}
	return entryRow{}, false, err
}

func entryInSource(
	ctx context.Context,
	tx *sql.Tx,
	entry channel.EntryID,
	source registry.ChannelKey,
) (entryRow, bool, error) {
	var row entryRow
	var item string
	err := tx.QueryRowContext(ctx, `
SELECT work_item_id
FROM channel_entries
WHERE id = $1
  AND channel_key = $2
FOR UPDATE`, entry.String(), source.String()).Scan(&item)
	if err == nil {
		row.id = entry
		row.workItem = workitem.ID(item)
		return row, true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return entryRow{}, false, nil
	}
	return entryRow{}, false, err
}

func oldestAvailableEntries(
	ctx context.Context,
	tx *sql.Tx,
	source registry.ChannelKey,
	limit int,
) ([]channel.EntryID, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT e.id
FROM channel_entries e
WHERE e.channel_key = $1
  AND NOT EXISTS (
    SELECT 1 FROM leases l WHERE l.channel_entry_id = e.id
  )
ORDER BY e.available_at, e.enqueued_at, e.id
LIMIT $2
FOR UPDATE`, source.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntryIDs(rows)
}

func scanEntryIDs(rows *sql.Rows) ([]channel.EntryID, error) {
	var ids []channel.EntryID
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, channel.EntryID(id))
	}
	return ids, rows.Err()
}

func moveEntries(
	ctx context.Context,
	tx *sql.Tx,
	entries []channel.EntryID,
	target registry.ChannelKey,
) error {
	for _, entry := range entries {
		if _, err := tx.ExecContext(ctx, `
UPDATE channel_entries
SET channel_key = $2
WHERE id = $1`, entry.String(), target.String()); err != nil {
			return err
		}
	}
	return nil
}

func deleteCurrentEntries(ctx context.Context, tx *sql.Tx, items []workitem.ID) error {
	for _, item := range items {
		if _, err := tx.ExecContext(ctx, `DELETE FROM channel_entries WHERE work_item_id = $1`, item.String()); err != nil {
			return err
		}
	}
	return nil
}
