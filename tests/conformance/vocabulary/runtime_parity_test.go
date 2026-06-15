package vocabulary_test

import (
	"testing"

	"github.com/pay-bye/agent-os/internal/declaration"
)

func TestRuntimeAcceptsSchemaAcceptedVocabulary(t *testing.T) {
	if _, err := declaration.Parse([]byte(validVocabulary())); err != nil {
		t.Fatalf("runtime rejected valid vocabulary: %v", err)
	}
}

func TestRuntimeRejectsSchemaRejectedVocabulary(t *testing.T) {
	for _, test := range invalidVocabularyCases() {
		t.Run(test.name, func(t *testing.T) {
			if _, err := declaration.Parse([]byte(test.body)); err == nil {
				t.Fatal("expected runtime rejection")
			}
		})
	}
}
