package registry

import "testing"

func TestNeedKindReportsOptionalSchema(t *testing.T) {
	withoutSchema := NewNeedKind(NeedKindKey("x12"), "First")
	withSchema := NewNeedKindWithSchema(NeedKindKey("x13"), SchemaKey("x61"), "Second")

	if withoutSchema.Key().String() != "x12" {
		t.Fatalf("need key = %q, want x12", withoutSchema.Key())
	}
	if withoutSchema.Description() != "First" {
		t.Fatalf("description = %q, want First", withoutSchema.Description())
	}
	if _, ok := withoutSchema.SchemaKey(); ok {
		t.Fatal("need kind without schema reported schema")
	}
	got, ok := withSchema.SchemaKey()
	if !ok || got != SchemaKey("x61") {
		t.Fatalf("schema key = %q, %v; want x61, true", got, ok)
	}
}
