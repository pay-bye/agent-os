package vocabulary_test

import "testing"

func TestIdentifierIsRelative(t *testing.T) {
	schema := readSchema(t)

	if schema.ID != "v1.schema.json" {
		t.Fatalf("schema id = %q, want v1.schema.json", schema.ID)
	}
}
