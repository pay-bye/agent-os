package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
)

//go:embed *.sql
var migrationFiles embed.FS

var ErrMigrationChecksumMismatch = errors.New("migration checksum mismatch")

type Migration struct {
	Name string
	Body []byte
}

type Option func(*Migrator)

type migrationPlan struct {
	Apply []Migration
}

type Migrator struct {
	db         *sql.DB
	searchPath string
	migrations []Migration
}

func (m Migrator) Apply(ctx context.Context) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	err = m.apply(ctx, tx)
	return finish(tx, err)
}

func (m Migrator) apply(ctx context.Context, tx *sql.Tx) error {
	if err := setSearchPath(ctx, tx, m.searchPath); err != nil {
		return err
	}
	if err := ensureLedger(ctx, tx); err != nil {
		return err
	}
	current, err := appliedMigrations(ctx, tx)
	if err != nil {
		return err
	}
	plan, err := planMigrations(m.migrations, current)
	if err != nil {
		return err
	}
	return applyMigrations(ctx, tx, plan.Apply)
}

func New(db *sql.DB, options ...Option) Migrator {
	migrator := Migrator{db: db, migrations: embeddedMigrations()}
	for _, option := range options {
		option(&migrator)
	}
	return migrator
}

func WithSource(migrations []Migration) Option {
	return func(migrator *Migrator) {
		migrator.migrations = append([]Migration(nil), migrations...)
	}
}

func WithSearchPath(path string) Option {
	return func(migrator *Migrator) {
		migrator.searchPath = path
	}
}

func planMigrations(migrations []Migration, current map[string]string) (migrationPlan, error) {
	ordered := append([]Migration(nil), migrations...)
	sort.Slice(ordered, func(left int, right int) bool {
		return ordered[left].Name < ordered[right].Name
	})
	plan := migrationPlan{}
	for _, migration := range ordered {
		sum := checksum(migration.Body)
		applied, ok := current[migration.Name]
		if !ok {
			plan.Apply = append(plan.Apply, migration)
			continue
		}
		if applied != sum {
			return migrationPlan{}, fmt.Errorf("%w: %s", ErrMigrationChecksumMismatch, migration.Name)
		}
	}
	return plan, nil
}

func ensureLedger(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS migrations (
  filename TEXT PRIMARY KEY CONSTRAINT migrations_filename_not_empty CHECK (length(btrim(filename)) > 0),
  checksum TEXT NOT NULL CONSTRAINT migrations_checksum_not_empty CHECK (length(btrim(checksum)) > 0),
  applied_at TIMESTAMPTZ NOT NULL
)`)
	return err
}

func appliedMigrations(ctx context.Context, tx *sql.Tx) (map[string]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT filename, checksum FROM migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := map[string]string{}
	for rows.Next() {
		var name string
		var sum string
		if err := rows.Scan(&name, &sum); err != nil {
			return nil, err
		}
		items[name] = sum
	}
	return items, rows.Err()
}

func applyMigrations(ctx context.Context, tx *sql.Tx, migrations []Migration) error {
	for _, migration := range migrations {
		if _, err := tx.ExecContext(ctx, string(migration.Body)); err != nil {
			return err
		}
		if err := recordMigration(ctx, tx, migration); err != nil {
			return err
		}
	}
	return nil
}

func recordMigration(ctx context.Context, tx *sql.Tx, migration Migration) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO migrations (filename, checksum, applied_at)
VALUES ($1, $2, CURRENT_TIMESTAMP)`, migration.Name, checksum(migration.Body))
	return err
}

func checksum(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func embeddedMigrations() []Migration {
	entries, err := fs.ReadDir(migrationFiles, ".")
	if err != nil {
		panic(err)
	}
	migrations := make([]Migration, 0, len(entries))
	for _, entry := range entries {
		migrations = append(migrations, readEmbeddedMigration(entry.Name()))
	}
	return migrations
}

func readEmbeddedMigration(name string) Migration {
	body, err := migrationFiles.ReadFile(filepath.ToSlash(name))
	if err != nil {
		panic(err)
	}
	return Migration{Name: name, Body: body}
}
