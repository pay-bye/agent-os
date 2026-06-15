//go:build integration

package postgresfixture

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Open(t testing.TB) *sql.DB {
	t.Helper()

	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	})
	return db
}

func MigratedTransaction(t testing.TB, ctx context.Context) *sql.Tx {
	t.Helper()

	db := Open(t)
	tx := Begin(t, ctx, db)
	SetSearchPath(t, ctx, tx, "pg_temp")
	ApplyMigrations(t, ctx, tx)
	return tx
}

func Begin(t testing.TB, ctx context.Context, db *sql.DB) *sql.Tx {
	t.Helper()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			t.Fatal(err)
		}
	})
	return tx
}

func CreateSchema(t testing.TB, ctx context.Context, db *sql.DB, prefix string) string {
	t.Helper()

	schema := schemaName(t, prefix)
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), "DROP SCHEMA IF EXISTS "+schema+" CASCADE"); err != nil {
			t.Fatal(err)
		}
	})
	return schema
}

func SetSearchPath(t testing.TB, ctx context.Context, tx *sql.Tx, schema string) {
	t.Helper()

	if _, err := tx.ExecContext(ctx, "SELECT set_config('search_path', $1, true)", schema); err != nil {
		t.Fatal(err)
	}
}

func ApplyMigrations(t testing.TB, ctx context.Context, tx *sql.Tx) {
	t.Helper()

	for _, path := range migrationPaths(t) {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx, string(migration)); err != nil {
			t.Fatal(err)
		}
	}
}

func MigrationPath(t testing.TB, name string) string {
	t.Helper()

	return filepath.Join(migrationDirectory(t), name)
}

func ReadMigration(t testing.TB, name string) []byte {
	t.Helper()

	migration, err := os.ReadFile(MigrationPath(t, name))
	if err != nil {
		t.Fatal(err)
	}
	return migration
}

func schemaName(t testing.TB, prefix string) string {
	t.Helper()

	if !validIdentifierPart(prefix) {
		t.Fatalf("schema prefix %q is not a safe identifier part", prefix)
	}
	return fmt.Sprintf("%s_%d_%d", prefix, os.Getpid(), time.Now().UnixNano())
}

func validIdentifierPart(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char != '_' && (char < 'a' || char > 'z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}

func migrationPaths(t testing.TB) []string {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(migrationDirectory(t), "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	return paths
}

func migrationDirectory(t testing.TB) string {
	t.Helper()

	return filepath.Join(findRoot(t), "internal", "storage", "postgres", "migrations")
}

func findRoot(t testing.TB) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if exists(filepath.Join(dir, "go.mod")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("source root not found")
		}
		dir = parent
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
