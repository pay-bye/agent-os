package kernel

import (
	"context"
	"database/sql"
)

type rowScanner interface {
	Scan(...any) error
}

type searchPathExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func setSearchPath(ctx context.Context, executor searchPathExecutor, path string) error {
	if path == "" {
		return nil
	}
	_, err := executor.ExecContext(ctx, "SELECT set_config('search_path', $1, true)", path)
	return err
}

func finish(tx *sql.Tx, err error) error {
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
