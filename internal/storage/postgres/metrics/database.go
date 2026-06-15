package metrics

import (
	"context"
	"database/sql"
)

type queryFunc func(context.Context, string, ...any) rowScanner

type rowReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type rowScanner interface {
	Scan(...any) error
}
