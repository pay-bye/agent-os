package postgres

import (
	"context"
	"database/sql"
)

type searchPathExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func SetSearchPath(ctx context.Context, executor searchPathExecutor, path string) error {
	if path == "" {
		return nil
	}
	_, err := executor.ExecContext(ctx, "SELECT set_config('search_path', $1, true)", path)
	return err
}
