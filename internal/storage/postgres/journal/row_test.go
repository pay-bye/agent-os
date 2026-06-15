package journal

import (
	"database/sql"
	"errors"
	"time"
)

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
		case *[]byte:
			*destination = append([]byte(nil), value.([]byte)...)
		case *string:
			*destination = value.(string)
		case *int:
			*destination = value.(int)
		case *int64:
			*destination = value.(int64)
		case *time.Time:
			*destination = value.(time.Time)
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
