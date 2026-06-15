package registry

import (
	"context"
	"testing"

	records "github.com/pay-bye/agent-os/internal/registry"
)

func TestFindJournalEventKindMapsOptionalSchema(t *testing.T) {
	reader := &Store{query: func(context.Context, string, ...any) rowScanner {
		return rowValues{values: []any{"x01", "First"}}
	}}

	item, err := reader.FindJournalEventKind(context.Background(), records.JournalEventKindKey("x20"))
	if err != nil {
		t.Fatal(err)
	}

	schema, ok := item.Schema()
	if !ok || schema != records.SchemaKey("x01") {
		t.Fatalf("schema = %q, %v; want x01, true", schema, ok)
	}
}
