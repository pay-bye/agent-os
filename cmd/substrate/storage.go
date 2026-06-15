package main

import (
	"context"
	"database/sql"
	"github.com/pay-bye/agent-os/internal/declaration/execution"
	"github.com/pay-bye/agent-os/internal/processlog"
	"github.com/pay-bye/agent-os/internal/storage/postgres/catalog"
	"github.com/pay-bye/agent-os/internal/storage/postgres/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type storage struct {
	recorder processlog.Recorder
}

func (s storage) Open(ctx context.Context, databaseURL string) (execution.Store, func() error, error) {
	db, err := openMigratedDatabase(ctx, databaseURL, s.recorder)
	if err != nil {
		return nil, nil, err
	}
	store, err := openStore(ctx, db)
	if err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	return store, db.Close, nil
}

func openMigratedDatabase(
	ctx context.Context,
	databaseURL string,
	recorders ...processlog.Recorder,
) (*sql.DB, error) {
	recorder := firstRecorder(recorders)
	db, err := openDatabase(databaseURL)
	if err != nil {
		record(recorder, processlog.StorageMigrated(processlog.Failed, processlog.StorageUnavailable))
		return nil, err
	}
	if err := migrate(ctx, db); err != nil {
		_ = db.Close()
		record(recorder, processlog.StorageMigrated(processlog.Failed, processlog.StorageMigration))
		return nil, err
	}
	record(recorder, processlog.StorageMigrated(processlog.Succeeded, ""))
	return db, nil
}

func openDatabase(databaseURL string) (*sql.DB, error) {
	return sql.Open("pgx", databaseURL)
}

func migrate(ctx context.Context, db *sql.DB) error {
	return migrations.New(db).Apply(ctx)
}

func openStore(ctx context.Context, db *sql.DB) (*catalog.Store, error) {
	return catalog.Open(ctx, db, "")
}

func firstRecorder(recorders []processlog.Recorder) processlog.Recorder {
	if len(recorders) == 0 {
		return nil
	}
	return recorders[0]
}
