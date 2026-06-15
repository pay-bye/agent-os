package registry

import "testing"

func TestNewJournalEventKindReportsFields(t *testing.T) {
	withoutSchema, err := NewJournalEventKind(JournalEventKindInput{
		Key:         JournalEventKindKey("x20"),
		Description: "First",
	})
	if err != nil {
		t.Fatal(err)
	}
	withSchema, err := NewJournalEventKind(JournalEventKindInput{
		Key:         JournalEventKindKey("x21"),
		Schema:      SchemaKey("x01"),
		HasSchema:   true,
		Description: "Second",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := withoutSchema.Schema(); ok {
		t.Fatal("event kind without schema reported schema")
	}
	schema, ok := withSchema.Schema()
	if !ok || schema != SchemaKey("x01") {
		t.Fatalf("schema = %q, %v; want x01, true", schema, ok)
	}
}
