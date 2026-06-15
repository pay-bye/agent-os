//go:build integration

package postgres_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"testing"

	migrationstore "github.com/pay-bye/agent-os/internal/storage/postgres/migrations"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

func TestMigratorRecordsLedgerAndSkipsUnchangedFiles(t *testing.T) {
	ctx := context.Background()
	schema := databaseWithSchema(t, ctx, "x93")
	migrations := []migrationstore.Migration{{
		Name: "001.sql",
		Body: []byte("CREATE TABLE observed (value TEXT); INSERT INTO observed (value) VALUES ('x01');"),
	}}
	migrator := schema.migrator(migrations)

	if err := migrator.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := migrator.Apply(ctx); err != nil {
		t.Fatal(err)
	}

	schema.requireLedger(t, ctx, migrations[0])
	schema.requireCount(t, ctx, rowCount{table: "observed", want: 1})
}

func TestMigratorRejectsChecksumDriftBeforeLaterFiles(t *testing.T) {
	ctx := context.Background()
	schema := databaseWithSchema(t, ctx, "x94")
	first := migrationstore.Migration{Name: "001.sql", Body: []byte("CREATE TABLE first_seen (value TEXT);")}
	later := migrationstore.Migration{Name: "002.sql", Body: []byte("CREATE TABLE later_seen (value TEXT);")}
	migrator := schema.migrator([]migrationstore.Migration{first})
	if err := migrator.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	schema.tamperChecksum(t, ctx, "001.sql")
	migrator = schema.migrator([]migrationstore.Migration{first, later})

	err := migrator.Apply(ctx)

	if !errors.Is(err, migrationstore.ErrMigrationChecksumMismatch) {
		t.Fatalf("error = %v, want checksum mismatch", err)
	}
	schema.requireTableMissing(t, ctx, "later_seen")
}

func TestMigratorRollsBackBodyAndLedgerWhenFileFails(t *testing.T) {
	ctx := context.Background()
	schema := databaseWithSchema(t, ctx, "x95")
	migrations := []migrationstore.Migration{
		{Name: "001.sql", Body: []byte("CREATE TABLE first_seen (value TEXT);")},
		{Name: "002.sql", Body: []byte("BROKEN SQL")},
	}
	migrator := schema.migrator(migrations)

	if err := migrator.Apply(ctx); err == nil {
		t.Fatal("expected migration failure")
	}
	schema.requireTableMissing(t, ctx, "first_seen")
	schema.requireTableMissing(t, ctx, "migrations")
}

type testSchema struct {
	db   *sql.DB
	name string
}

type rowCount struct {
	table string
	want  int
}

func databaseWithSchema(t *testing.T, ctx context.Context, prefix string) testSchema {
	t.Helper()

	db := postgresfixture.Open(t)
	schema := postgresfixture.CreateSchema(t, ctx, db, prefix)
	return testSchema{db: db, name: schema}
}

func (schema testSchema) migrator(migrations []migrationstore.Migration) migrationstore.Migrator {
	return migrationstore.New(
		schema.db,
		migrationstore.WithSearchPath(schema.name),
		migrationstore.WithSource(migrations),
	)
}

func (schema testSchema) tamperChecksum(t *testing.T, ctx context.Context, filename string) {
	t.Helper()

	_, err := schema.db.ExecContext(ctx, "UPDATE "+schema.table("migrations")+" SET checksum = 'different' WHERE filename = $1", filename)
	if err != nil {
		t.Fatal(err)
	}
}

func (schema testSchema) requireLedger(t *testing.T, ctx context.Context, migration migrationstore.Migration) {
	t.Helper()

	var got string
	err := schema.db.QueryRowContext(
		ctx,
		"SELECT checksum FROM "+schema.table("migrations")+" WHERE filename = $1",
		migration.Name,
	).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	want := checksum(migration.Body)
	if got != want {
		t.Fatalf("checksum = %q, want %q", got, want)
	}
}

func (schema testSchema) requireCount(t *testing.T, ctx context.Context, expected rowCount) {
	t.Helper()

	var count int
	if err := schema.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+schema.table(expected.table)).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != expected.want {
		t.Fatalf("%s count = %d, want %d", expected.table, count, expected.want)
	}
}

func (schema testSchema) requireTableMissing(t *testing.T, ctx context.Context, table string) {
	t.Helper()

	var exists bool
	err := schema.db.QueryRowContext(ctx, "SELECT to_regclass($1) IS NOT NULL", schema.table(table)).Scan(&exists)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatalf("expected %s to be absent", schema.table(table))
	}
}

func (schema testSchema) table(name string) string {
	return schema.name + "." + name
}

func checksum(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
