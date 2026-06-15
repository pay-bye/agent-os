package registry

import "testing"

func TestItemKindReportsFields(t *testing.T) {
	item := NewItemKind(ItemKindKey("x08"), SchemaKey("x01"), "First")

	if item.Key().String() != "x08" {
		t.Fatalf("item key = %q, want x08", item.Key())
	}
	if item.SchemaKey().String() != "x01" {
		t.Fatalf("schema key = %q, want x01", item.SchemaKey())
	}
	if item.Description() != "First" {
		t.Fatalf("description = %q, want First", item.Description())
	}
}
