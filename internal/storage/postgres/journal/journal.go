package journal

import (
	"context"
	"database/sql"
	"errors"
	eventlog "github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
	"time"
)

type journaler interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type eventInput struct {
	id          string
	coordinate  string
	key         string
	kind        string
	appendedAt  time.Time
	appendIndex int64
	payload     []byte
}

type Store struct {
	command   commandFunc
	query     queryFunc
	queryRows rowsQueryFunc
}

func (j *Store) Append(ctx context.Context, input eventlog.EventInput) (eventlog.Event, error) {
	event, err := eventlog.NewEvent(input)
	if err != nil {
		return eventlog.Event{}, err
	}
	if err := j.requireEventKind(ctx, event.Kind()); err != nil {
		return eventlog.Event{}, err
	}
	var appendIndex int64
	err = j.query(ctx, `
	INSERT INTO journal_events (id, coordinate_kind, coordinate_key, event_kind_key, appended_at, payload)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING append_index`,
		event.ID().String(),
		string(event.Coordinate().Kind()),
		event.Coordinate().Key(),
		event.Kind().String(),
		event.AppendedAt(),
		event.Payload(),
	).Scan(&appendIndex)
	if err != nil {
		return eventlog.Event{}, err
	}
	return eventlog.NewRecordedEvent(input, appendIndex)
}

func (j *Store) requireEventKind(ctx context.Context, key registry.JournalEventKindKey) error {
	var found int
	err := j.query(ctx, `SELECT 1 FROM journal_event_kinds WHERE key = $1`, key.String()).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return registry.JournalEventKindNotFound(key)
	}
	return err
}

func (j *Store) Replay(ctx context.Context, id workitem.ID) ([]eventlog.Event, error) {
	return j.replay(ctx, eventlog.WorkItemCoordinate(id))
}

func (j *Store) replay(ctx context.Context, coordinate eventlog.Coordinate) ([]eventlog.Event, error) {
	rows, err := j.queryRows(ctx, `
	SELECT id, coordinate_kind, coordinate_key, event_kind_key, appended_at, append_index, payload
	FROM journal_events
	WHERE coordinate_kind = $1
	  AND coordinate_key = $2
	ORDER BY append_index`, string(coordinate.Kind()), coordinate.Key())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []eventlog.Event
	for rows.Next() {
		event, err := ScanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (j *Store) ReplayNode(ctx context.Context, node registry.NodeKey) ([]eventlog.Event, error) {
	return j.replay(ctx, eventlog.NodeCoordinate(node))
}

func New(db journaler) *Store {
	return &Store{
		command: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			return db.ExecContext(ctx, query, args...)
		},
		query: func(ctx context.Context, query string, args ...any) rowScanner {
			return db.QueryRowContext(ctx, query, args...)
		},
		queryRows: func(ctx context.Context, query string, args ...any) (rowsScanner, error) {
			return db.QueryContext(ctx, query, args...)
		},
	}
}

func ScanEvent(row rowScanner) (eventlog.Event, error) {
	var input eventInput
	if err := row.Scan(
		&input.id,
		&input.coordinate,
		&input.key,
		&input.kind,
		&input.appendedAt,
		&input.appendIndex,
		&input.payload,
	); err != nil {
		return eventlog.Event{}, err
	}
	coordinate, err := eventlog.NewCoordinate(eventlog.CoordinateKind(input.coordinate), input.key)
	if err != nil {
		return eventlog.Event{}, err
	}
	return eventlog.NewRecordedEvent(eventlog.EventInput{
		ID:         eventlog.EventID(input.id),
		Coordinate: coordinate,
		Kind:       registry.JournalEventKindKey(input.kind),
		AppendedAt: input.appendedAt,
		Payload:    input.payload,
	}, input.appendIndex)
}
