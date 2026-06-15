package channel

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

type missingRow struct{}

func (missingRow) Scan(...any) error {
	return sql.ErrNoRows
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
		case *string:
			*destination = value.(string)
		case *time.Time:
			*destination = value.(time.Time)
		case *sql.NullTime:
			timestamp, ok := value.(time.Time)
			*destination = sql.NullTime{Time: timestamp, Valid: ok}
		default:
			return errors.New("unsupported destination")
		}
	}
	return nil
}
