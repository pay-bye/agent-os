package postgres

import (
	"context"
	"database/sql"
	"testing"
)

func TestSetSearchPathSkipsEmptyValue(t *testing.T) {
	executor := &recordingExecutor{}

	err := SetSearchPath(context.Background(), executor, "")

	if err != nil {
		t.Fatal(err)
	}
	if len(executor.statements) != 0 {
		t.Fatalf("statements = %v", executor.statements)
	}
}

func TestSetSearchPathAppliesLocalScope(t *testing.T) {
	executor := &recordingExecutor{}

	err := SetSearchPath(context.Background(), executor, "x01")

	if err != nil {
		t.Fatal(err)
	}
	requireStatementOrder(t, executor.statements, []string{"SELECT set_config('search_path', $1, true)"})
	requireArgs(t, executor.args[0], "x01")
}

func requireStatementOrder(t *testing.T, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("statements = %v, want %v", got, want)
	}
	for index := range got {
		if got[index] != want[index] {
			t.Fatalf("statement[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

type recordingExecutor struct {
	statements []string
	args       [][]any
}

func (e *recordingExecutor) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	copied := append([]any(nil), args...)
	e.statements = append(e.statements, query)
	e.args = append(e.args, copied)
	return nil, nil
}
