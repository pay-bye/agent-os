package postgres

import (
	"context"
	"database/sql"
)

type queryFunc func(context.Context, string, ...any) rowScanner

type rowsQueryFunc func(context.Context, string, ...any) (rowsScanner, error)

type rowReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type rowScanner interface {
	Scan(...any) error
}

type rowsScanner interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close() error
}
