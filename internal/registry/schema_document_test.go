package registry

import "testing"

func TestSchemaDocumentProtectsDocumentBytes(t *testing.T) {
	document := []byte(`{"title":"First"}`)
	item := NewSchemaDocument(SchemaKey("x01"), document)

	document[0] = '['
	got := item.Document()
	got[0] = '['

	if string(item.Document()) != `{"title":"First"}` {
		t.Fatalf("schema document must not expose mutable payload alias")
	}
	if item.Key() != SchemaKey("x01") {
		t.Fatalf("schema key = %q, want x01", item.Key())
	}
}

func TestSchemaDocumentAllowsAbsentDocument(t *testing.T) {
	item := NewSchemaDocument(SchemaKey("x62"), nil)

	if item.Document() != nil {
		t.Fatal("absent document must remain absent")
	}
}
