package declaration

import (
	"encoding/json"
	"testing"
)

func TestParseAcceptsPositiveVocabulary(t *testing.T) {
	document, err := Parse([]byte(validDocument()))
	if err != nil {
		t.Fatal(err)
	}

	if document.Version != 1 {
		t.Fatalf("version = %d", document.Version)
	}
	if _, ok := document.Schemas["x01"]; !ok {
		t.Fatal("schema missing")
	}
}

func TestParsePreservesSchemaJSONTypes(t *testing.T) {
	document := mustParse(t, `version: 1
schemas:
  x01:
    document:
      type: object
      required:
        - x40
      additionalProperties: false
      minimum: 1
      exclusive: true
      ratio: 1.5
      nullable: null
items: {}
needs: {}
nodes: {}
routes: {}
`)

	var schema map[string]any
	if err := json.Unmarshal(document.Schemas["x01"].Document, &schema); err != nil {
		t.Fatal(err)
	}

	required := schema["required"].([]any)
	if required[0] != "x40" || schema["exclusive"] != true || schema["nullable"] != nil {
		t.Fatalf("schema = %+v", schema)
	}
	if schema["minimum"] != float64(1) || schema["ratio"] != 1.5 {
		t.Fatalf("schema numbers = %+v", schema)
	}
}
