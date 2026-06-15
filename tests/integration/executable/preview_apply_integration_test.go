//go:build integration

package executable_test

import (
	"bytes"
	"context"
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/pay-bye/agent-os/internal/declaration"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

const verifierDigest = "47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU"

func TestPreviewAndApplyUseExplicitVocabularyTransaction(t *testing.T) {
	ctx := context.Background()
	db := postgresfixture.Open(t)
	schema := postgresfixture.CreateSchema(t, ctx, db, "x91")
	binary := buildBinary(t)
	input := deltaInput{
		databaseURL: withSearchPath(t, os.Getenv("DATABASE_URL"), schema),
		declaration: writeVocabulary(t),
	}

	previewed := runDeltaCommand(t, binary, "preview", input)
	requireAdditions(t, previewed)
	requireSchemaCount(t, ctx, db, schema, 0)

	applied := runDeltaCommand(t, binary, "apply", input)
	requireAdditions(t, applied)
	requireSchemaCount(t, ctx, db, schema, 1)

	steady := runDeltaCommand(t, binary, "preview", input)
	if len(steady.Additions) != 0 || len(steady.Conflicts) != 0 {
		t.Fatalf("steady delta = %+v", steady)
	}
}

func TestApplyClearsExclusionsAndRemovesVocabulary(t *testing.T) {
	ctx := context.Background()
	db := postgresfixture.Open(t)
	schema := postgresfixture.CreateSchema(t, ctx, db, "x93")
	binary := buildBinary(t)
	input := deltaInput{
		databaseURL: withSearchPath(t, os.Getenv("DATABASE_URL"), schema),
		declaration: writeExpandedVocabulary(t),
	}
	runDeltaCommand(t, binary, "apply", input)
	insertRoutingExclusion(t, ctx, db, schema, "x17")
	insertRoutingExclusion(t, ctx, db, schema, "x18")
	input.declaration = writeVocabulary(t)

	previewed := runDeltaCommand(t, binary, "preview", input)
	applied := runDeltaCommand(t, binary, "apply", input)

	requireRef(t, previewed.Clearances, "routing_exclusion", "x17")
	requireRef(t, previewed.Removals, "node", "x18")
	requireRef(t, applied.Clearances, "routing_exclusion", "x17")
	requireNodeAbsent(t, ctx, db, schema, "x18")
	requireExclusionCount(t, ctx, db, schema, 0)
	requireExclusionClearEvent(t, ctx, db, schema, "x17")
	requireExclusionClearEvent(t, ctx, db, schema, "x18")
}

func TestApplyFailsClosedWhenRemovalHasLiveEntry(t *testing.T) {
	ctx := context.Background()
	db := postgresfixture.Open(t)
	schema := postgresfixture.CreateSchema(t, ctx, db, "x94")
	binary := buildBinary(t)
	input := deltaInput{
		databaseURL: withSearchPath(t, os.Getenv("DATABASE_URL"), schema),
		declaration: writeExpandedVocabulary(t),
	}
	runDeltaCommand(t, binary, "apply", input)
	insertLiveEntry(t, ctx, db, schema, "x18")
	input.declaration = writeVocabulary(t)

	runFailingDeltaCommand(t, binary, "apply", input)

	requireNodePresent(t, ctx, db, schema, "x18")
}

func TestServeAppliesMigrationsAndFailsOnDeclarationDriftBeforeBind(t *testing.T) {
	ctx := context.Background()
	db := postgresfixture.Open(t)
	schema := postgresfixture.CreateSchema(t, ctx, db, "x92")

	requireServeDeclarationDrift(t, serveInput{
		schema:         schema,
		declaration:    writeVocabulary(t),
		address:        "127.0.0.1:1",
		verifierDigest: verifierDigest,
	})
	requireMigrationCount(t, ctx, db, schema)
}

func requireServeDeclarationDrift(t *testing.T, input serveInput) {
	t.Helper()

	var output bytes.Buffer
	command := serveCommand(t, input)
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	if err == nil {
		t.Fatal("serve succeeded with declaration drift")
	}
	requireContains(t, output.String(), "declaration_drift")
}

func writeVocabulary(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "vocabulary.yaml")
	content := []byte(`version: 1
schemas:
  x01:
    document:
      type: object
items:
  x08:
    schema: x01
    description: x21
needs:
  x12:
    schema: x01
    description: x22
nodes:
  x17:
    description: x23
    accepts:
      - x12
routes:
  x12:
    - node: x17
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeExpandedVocabulary(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "vocabulary.yaml")
	content := []byte(`version: 1
schemas:
  x01:
    document:
      type: object
items:
  x08:
    schema: x01
    description: x21
needs:
  x12:
    schema: x01
    description: x22
nodes:
  x17:
    description: x23
    accepts:
      - x12
  x18:
    description: x24
    accepts:
      - x12
routes:
  x12:
    - node: x17
    - node: x18
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func withSearchPath(t *testing.T, databaseURL string, schema string) string {
	t.Helper()

	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("options", "-c search_path="+schema)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func insertRoutingExclusion(t *testing.T, ctx context.Context, db *sql.DB, schema string, node string) {
	t.Helper()

	_, err := db.ExecContext(ctx, `INSERT INTO `+schema+`.routing_exclusions (node_key) VALUES ($1)`, node)
	if err != nil {
		t.Fatal(err)
	}
}

func insertLiveEntry(t *testing.T, ctx context.Context, db *sql.DB, schema string, node string) {
	t.Helper()

	_, err := db.ExecContext(ctx, `
INSERT INTO `+schema+`.channel_entries (id, channel_key, work_item_id, enqueued_at, available_at)
SELECT 'x32', c.key, 'x08', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM `+schema+`.channels c
WHERE c.node_key = $1`, node)
	if err != nil {
		t.Fatal(err)
	}
}

func requireRef(t *testing.T, refs []declaration.RecordRef, kind string, key string) {
	t.Helper()

	for _, ref := range refs {
		if ref.Kind == kind && ref.Key == key {
			return
		}
	}
	t.Fatalf("refs = %+v, want %s %s", refs, kind, key)
}

func requireNodeAbsent(t *testing.T, ctx context.Context, db *sql.DB, schema string, node string) {
	t.Helper()

	requireNodeCount(t, ctx, db, schema, node, 0)
}

func requireNodePresent(t *testing.T, ctx context.Context, db *sql.DB, schema string, node string) {
	t.Helper()

	requireNodeCount(t, ctx, db, schema, node, 1)
}

func requireNodeCount(t *testing.T, ctx context.Context, db *sql.DB, schema string, node string, want int) {
	t.Helper()

	var got int
	err := db.QueryRowContext(ctx, `SELECT count(*) FROM `+schema+`.nodes WHERE key = $1`, node).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("node count = %d, want %d", got, want)
	}
}

func requireExclusionCount(t *testing.T, ctx context.Context, db *sql.DB, schema string, want int) {
	t.Helper()

	var got int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM `+schema+`.routing_exclusions`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("exclusion count = %d, want %d", got, want)
	}
}

func requireExclusionClearEvent(t *testing.T, ctx context.Context, db *sql.DB, schema string, node string) {
	t.Helper()

	var got int
	err := db.QueryRowContext(ctx, `
SELECT count(*)
FROM `+schema+`.journal_events
WHERE coordinate_kind = 'node'
  AND coordinate_key = $1
  AND event_kind_key = 'x46'
  AND payload->>'node_key' = $1`, node).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("exclusion clear events for %s = %d, want 1", node, got)
	}
}

func requireAdditions(t *testing.T, delta declaration.Delta) {
	t.Helper()

	if !delta.Installable || len(delta.Additions) == 0 {
		t.Fatalf("delta = %+v", delta)
	}
}

func requireMigrationCount(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
	t.Helper()

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+schema+".migrations").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected migration ledger rows")
	}
}

func requireSchemaCount(t *testing.T, ctx context.Context, db *sql.DB, schema string, want int) {
	t.Helper()

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+schema+".schema_documents").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("schema_documents count = %d, want %d", count, want)
	}
}
