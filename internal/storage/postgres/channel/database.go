package channel

import (
	"context"
	"database/sql"
)

type commandFunc func(context.Context, string, ...any) (sql.Result, error)

type queryFunc func(context.Context, string, ...any) rowScanner

type rowScanner interface {
	Scan(...any) error
}
