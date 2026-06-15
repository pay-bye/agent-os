package declaration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitWritesOnlyDeclarationFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vocabulary.yaml")

	err := Init(InitInput{Path: path})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Parse(mustRead(t, path)); err != nil {
		t.Fatal(err)
	}
	items, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name() != "vocabulary.yaml" {
		t.Fatalf("entries = %v", items)
	}
}

func TestInitRejectsImplicitConfirmedInput(t *testing.T) {
	err := Init(InitInput{Yes: true})

	if err != ErrMissingExplicitVocabularyInput {
		t.Fatalf("error = %v, want missing explicit input", err)
	}
}

func TestInitRejectsExistingDeclaration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vocabulary.yaml")
	if err := Init(InitInput{Path: path}); err != nil {
		t.Fatal(err)
	}

	err := Init(InitInput{Path: path})

	requireError(t, err, "declaration_exists")
}
