package postgres

import (
	"database/sql"
	"errors"
	"testing"
	"time"
)

func requireQuery(t *testing.T, got string, want string) {
	t.Helper()

	if got != want {
		t.Fatalf("query = %q, want %q", got, want)
	}
}

func requireArgs(t *testing.T, got []any, want ...any) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("args = %v, want %v", got, want)
	}
	for index := range got {
		if got[index] != want[index] {
			t.Fatalf("arg[%d] = %v, want %v", index, got[index], want[index])
		}
	}
}

func instant(offset int) time.Time {
	return time.Date(2026, 5, 18, 12, offset, 0, 0, time.UTC)
}

type rowValues struct {
	values []any
}

func (r rowValues) Scan(destinations ...any) error {
	if len(destinations) != len(r.values) {
		return errors.New("destination count mismatch")
	}
	for index, value := range r.values {
		switch destination := destinations[index].(type) {
		case *[]byte:
			*destination = append([]byte(nil), value.([]byte)...)
		case *string:
			*destination = value.(string)
		case *int:
			*destination = value.(int)
		case *int64:
			*destination = value.(int64)
		case *bool:
			*destination = value.(bool)
		case *time.Time:
			*destination = value.(time.Time)
		case *sql.NullString:
			text, ok := value.(string)
			*destination = sql.NullString{String: text, Valid: ok}
		case *sql.NullInt64:
			number, ok := value.(int64)
			*destination = sql.NullInt64{Int64: number, Valid: ok}
		case *sql.NullTime:
			timestamp, ok := value.(time.Time)
			*destination = sql.NullTime{Time: timestamp, Valid: ok}
		default:
			return errors.New("unsupported destination")
		}
	}
	return nil
}

type rowsValues struct {
	rows  [][]any
	index int
}

func (r *rowsValues) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}

func (r *rowsValues) Scan(destinations ...any) error {
	return rowValues{values: r.rows[r.index-1]}.Scan(destinations...)
}

func (r *rowsValues) Err() error {
	return nil
}

func (r *rowsValues) Close() error {
	return nil
}
