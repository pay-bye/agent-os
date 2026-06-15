package registry

import (
	"context"
	"testing"

	records "github.com/pay-bye/agent-os/internal/registry"
)

func TestFindNeedKindMapsAbsentSchema(t *testing.T) {
	reader := &Store{query: func(context.Context, string, ...any) rowScanner {
		return rowValues{values: []any{nil, "First"}}
	}}

	item, err := reader.FindNeedKind(context.Background(), records.NeedKindKey("x12"))
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := item.SchemaKey(); ok {
		t.Fatal("need kind without schema reported schema")
	}
	if item.Description() != "First" {
		t.Fatalf("description = %q, want First", item.Description())
	}
}

func TestNeedKindFromRowMapsOptionalSchema(t *testing.T) {
	item, err := needKindFromRow(
		records.NeedKindKey("x12"),
		rowValues{values: []any{"x01", "First"}},
	)
	if err != nil {
		t.Fatal(err)
	}

	schemaKey, ok := item.SchemaKey()
	if !ok || schemaKey != records.SchemaKey("x01") {
		t.Fatalf("schema key = %q, %v; want x01, true", schemaKey, ok)
	}
	if item.Description() != "First" {
		t.Fatalf("description = %q, want First", item.Description())
	}
}
