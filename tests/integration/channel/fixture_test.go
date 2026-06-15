//go:build integration

package channel_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	channelstore "github.com/pay-bye/agent-os/internal/storage/postgres/channel"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

func setupSchema(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
	t.Helper()

	tx := postgresfixture.Begin(t, ctx, db)
	postgresfixture.SetSearchPath(t, ctx, tx, schema)
	postgresfixture.ApplyMigrations(t, ctx, tx)
	insertVocabulary(t, ctx, tx)
	store := channelstore.New(tx)
	enqueue(t, ctx, store, "x32", "x08", instant(0))
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func insertVocabulary(t *testing.T, ctx context.Context, tx *sql.Tx) {
	t.Helper()

	_, err := tx.ExecContext(ctx, `
INSERT INTO schema_documents (key, document) VALUES ('x01', '{"title":"First"}');
INSERT INTO need_kinds (key, schema_key, description) VALUES ('x12', 'x01', 'First');
INSERT INTO nodes (key, description) VALUES ('x17', 'First');
INSERT INTO channels (key, node_key, description) VALUES ('x15', 'x17', 'First');
INSERT INTO node_capabilities (node_key, need_kind_key) VALUES ('x17', 'x12');
`)
	if err != nil {
		t.Fatal(err)
	}
}

func instant(offset int) time.Time {
	return time.Date(2026, 5, 18, 12, offset, 0, 0, time.UTC)
}
