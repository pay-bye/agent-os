package vocabulary_test

import "testing"

func TestSchemaAcceptsVocabulary(t *testing.T) {
	schema := readSchema(t)

	if err := validate(schema, yamlValue(t, validVocabulary())); err != nil {
		t.Fatalf("schema rejected valid vocabulary: %v", err)
	}
}

func TestSchemaRejectsInvalidVocabulary(t *testing.T) {
	schema := readSchema(t)

	for _, test := range invalidVocabularyCases() {
		t.Run(test.name, func(t *testing.T) {
			if err := validate(schema, yamlValue(t, test.body)); err == nil {
				t.Fatal("expected schema rejection")
			}
		})
	}
}
