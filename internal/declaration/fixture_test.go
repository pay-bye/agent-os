package declaration

import (
	"os"
	"strings"
	"testing"
)

func validDocument() string {
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

func mustParse(t *testing.T, body string) Document {
	t.Helper()

	document, err := Parse([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func requireError(t *testing.T, err error, text string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error containing %q", text)
	}
	if !strings.Contains(err.Error(), text) {
		t.Fatalf("error = %v, want %q", err, text)
	}
}
