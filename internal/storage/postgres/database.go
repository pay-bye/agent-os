package postgres

import (
	"context"
	"database/sql"

	channelstore "github.com/pay-bye/agent-os/internal/storage/postgres/channel"
	journalstore "github.com/pay-bye/agent-os/internal/storage/postgres/journal"
	kernelstore "github.com/pay-bye/agent-os/internal/storage/postgres/kernel"
	metricstore "github.com/pay-bye/agent-os/internal/storage/postgres/metrics"
	migrationstore "github.com/pay-bye/agent-os/internal/storage/postgres/migrations"
	registrystore "github.com/pay-bye/agent-os/internal/storage/postgres/registry"
)

var ErrMigrationChecksumMismatch = migrationstore.ErrMigrationChecksumMismatch

type Kernel = kernelstore.Store

func NewKernel(db *sql.DB, options ...KernelOption) *Kernel {
	return kernelstore.New(db, options...)
}

type KernelOption = kernelstore.Option

type Migration = migrationstore.Migration

type Migrator = migrationstore.Migrator

func NewMigrator(db *sql.DB, options ...MigratorOption) Migrator {
	return migrationstore.New(db, options...)
}

type MigratorOption = migrationstore.Option

type commandReader interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type commandRowsReader interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type rowReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func WithSearchPath(path string) KernelOption {
	return kernelstore.WithSearchPath(path)
}

func WithMigrationSource(migrations []Migration) MigratorOption {
	return migrationstore.WithSource(migrations)
}

func WithMigratorSearchPath(path string) MigratorOption {
	return migrationstore.WithSearchPath(path)
}

func NewChannel(db commandReader) *channelstore.Store {
	return channelstore.New(db)
}

func NewJournal(db commandRowsReader) *journalstore.Store {
	return journalstore.New(db)
}

func NewRegistry(db rowReader) *registrystore.Store {
	return registrystore.New(db)
}

func NewMetrics(db rowReader) *metricstore.Store {
	return metricstore.New(db)
}
