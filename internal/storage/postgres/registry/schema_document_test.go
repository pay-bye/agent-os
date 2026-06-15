package registry

import (
	"context"
	"testing"

	records "github.com/pay-bye/agent-os/internal/registry"
)

func TestFindSchemaDocumentUsesReadQuery(t *testing.T) {
	reader := &Store{query: func(_ context.Context, query string, args ...any) rowScanner {
		requireQuery(t, query, `SELECT document FROM schema_documents WHERE key = $1`)
		requireArgs(t, args, "x01")
		return rowValues{values: []any{[]byte(`{"title":"First"}`)}}
	}}

	item, err := reader.FindSchemaDocument(context.Background(), records.SchemaKey("x01"))
	if err != nil {
		t.Fatal(err)
	}

	if string(item.Document()) != `{"title":"First"}` {
		t.Fatalf("document = %s, want Alpha document", item.Document())
	}
}

func TestSchemaDocumentFromRowReturnsNotFound(t *testing.T) {
	_, err := schemaDocumentFromRow(records.SchemaKey("x63"), missingRow{})

	if !records.IsNotFound(err) {
		t.Fatalf("expected registry not-found error, got %v", err)
	}
}
