package journal

import (
	"context"
	"database/sql"
)

type commandFunc func(context.Context, string, ...any) (sql.Result, error)

type queryFunc func(context.Context, string, ...any) rowScanner

type rowsQueryFunc func(context.Context, string, ...any) (rowsScanner, error)

type rowScanner interface {
	Scan(...any) error
}

type rowsScanner interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close() error
}
