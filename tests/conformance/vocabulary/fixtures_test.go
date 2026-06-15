package vocabulary_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type invalidVocabularyCase struct {
	name string
	body string
}

func invalidVocabularyCases() []invalidVocabularyCase {
	return []invalidVocabularyCase{
		{name: "invalid key", body: strings.Replace(validVocabulary(), "x12:", "_x12:", 1)},
		{name: "empty description", body: strings.Replace(validVocabulary(), "description: x22", "description: '   '", 1)},
		{name: "unknown field", body: strings.Replace(validVocabulary(), "description: x21", "description: x21\n    extra: x99", 1)},
		{name: "duplicate accepts", body: strings.Replace(validVocabulary(), "      - x12", "      - x12\n      - x12", 1)},
	}
}

func validVocabulary() string {
	return `version: 1
schemas:
  x01:
    document:
      type: object
items:
  x08:
    schema: x01
    description: x21
needs:
  x12:
    schema: x01
    description: x22
nodes:
  x17:
    description: x23
    accepts:
      - x12
routes:
  x12:
    - node: x17
`
}

func yamlValue(t *testing.T, body string) any {
	t.Helper()

	var value any
	if err := yaml.Unmarshal([]byte(body), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func findRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if exists(filepath.Join(dir, "go.mod")) && exists(filepath.Join(dir, "contracts", "vocabulary")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("source root not found")
		}
		dir = parent
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !errors.Is(err, os.ErrNotExist)
}
