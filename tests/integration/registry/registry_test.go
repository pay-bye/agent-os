//go:build integration

package registry_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/pay-bye/agent-os/internal/registry"
	registrystore "github.com/pay-bye/agent-os/internal/storage/postgres/registry"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

func TestReaderFindsRegistryRecords(t *testing.T) {
	ctx := context.Background()
	tx := migratedTransaction(t, ctx)
	insertRegistryRecords(t, ctx, tx)
	reader := registrystore.New(tx)

	requireRegistryRecords(t, ctx, reader)
}

func TestReaderFindsRegistryRecordsInFreshSchema(t *testing.T) {
	ctx := context.Background()
	tx := migratedFreshSchemaTransaction(t, ctx)
	insertRegistryRecords(t, ctx, tx)
	reader := registrystore.New(tx)

	requireRegistryRecords(t, ctx, reader)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func requireRegistryRecords(t *testing.T, ctx context.Context, reader *registrystore.Store) {
	t.Helper()

	requireSchemaDocument(t, ctx, reader)
	requireItemKind(t, ctx, reader)
	requireNeedKind(t, ctx, reader)
}

func requireSchemaDocument(t *testing.T, ctx context.Context, reader *registrystore.Store) {
	t.Helper()

	schema, err := reader.FindSchemaDocument(ctx, registry.SchemaKey("x01"))
	if err != nil {
		t.Fatal(err)
	}
	requireJSONField(t, schema.Document(), "title", "First")
}

func requireItemKind(t *testing.T, ctx context.Context, reader *registrystore.Store) {
	t.Helper()

	item, err := reader.FindItemKind(ctx, registry.ItemKindKey("x08"))
	if err != nil {
		t.Fatal(err)
	}
	if item.SchemaKey() != registry.SchemaKey("x01") {
		t.Fatalf("item schema key = %q, want x01", item.SchemaKey())
	}
	if item.Description() != "First" {
		t.Fatalf("item description = %q, want First", item.Description())
	}
}

func requireNeedKind(t *testing.T, ctx context.Context, reader *registrystore.Store) {
	t.Helper()

	need, err := reader.FindNeedKind(ctx, registry.NeedKindKey("x12"))
	if err != nil {
		t.Fatal(err)
	}
	schemaKey, ok := need.SchemaKey()
	if !ok || schemaKey != registry.SchemaKey("x01") {
		t.Fatalf("need schema key = %q, %v; want x01, true", schemaKey, ok)
	}
	if need.Description() != "First" {
		t.Fatalf("need description = %q, want First", need.Description())
	}
}

func TestReaderReturnsNamedNotFound(t *testing.T) {
	ctx := context.Background()
	tx := migratedTransaction(t, ctx)
	reader := registrystore.New(tx)

	_, err := reader.FindNeedKind(ctx, registry.NeedKindKey("x14"))

	if !registry.IsNotFound(err) {
		t.Fatalf("expected registry not-found error, got %v", err)
	}
}

func migratedTransaction(t *testing.T, ctx context.Context) *sql.Tx {
	t.Helper()

	db := postgresfixture.Open(t)
	tx := postgresfixture.Begin(t, ctx, db)
	postgresfixture.SetSearchPath(t, ctx, tx, "pg_temp")
	applyMigration(t, ctx, tx)

	return tx
}

func migratedFreshSchemaTransaction(t *testing.T, ctx context.Context) *sql.Tx {
	t.Helper()

	db := postgresfixture.Open(t)
	schema := postgresfixture.CreateSchema(t, ctx, db, "migration_check")
	tx := postgresfixture.Begin(t, ctx, db)
	postgresfixture.SetSearchPath(t, ctx, tx, schema)
	applyMigration(t, ctx, tx)

	return tx
}

func applyMigration(t *testing.T, ctx context.Context, tx *sql.Tx) {
	t.Helper()

	migration := postgresfixture.ReadMigration(t, "001_registry_vocabulary.sql")
	if _, err := tx.ExecContext(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
}

func insertRegistryRecords(t *testing.T, ctx context.Context, tx *sql.Tx) {
	t.Helper()

	_, err := tx.ExecContext(ctx, `
INSERT INTO schema_documents (key, document) VALUES ('x01', '{"title":"First"}');
INSERT INTO item_kinds (key, schema_key, description) VALUES ('x08', 'x01', 'First');
INSERT INTO need_kinds (key, schema_key, description) VALUES ('x12', 'x01', 'First');
`)
	if err != nil {
		t.Fatal(err)
	}
}

func requireJSONField(t *testing.T, document []byte, key string, want string) {
	t.Helper()

	var values map[string]string
	if err := json.Unmarshal(document, &values); err != nil {
		t.Fatal(err)
	}
	if values[key] != want {
		t.Fatalf("document[%q] = %q, want %q", key, values[key], want)
	}
}
