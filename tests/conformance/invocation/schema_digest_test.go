package invocation_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

const expectedSchemaSetDigest = "sha256:e08a34f1c973fc0f52336bb66686ceb9c70a0073080aa7fd53ce6a1e22be0325"

func TestSchemaSetDigestMatchesExpectedProjection(t *testing.T) {
	digest, err := schemaSetDigest(contractRoot(t))
	if err != nil {
		t.Fatal(err)
	}

	if digest != expectedSchemaSetDigest {
		t.Fatalf("schema set digest = %q, want %q", digest, expectedSchemaSetDigest)
	}
}

func TestSchemaSetDigestRejectsReferencedSchemaDrift(t *testing.T) {
	root := copyContract(t)
	before, err := schemaSetDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	replaceText(t, root, textEdit{
		path: schemaPath("ack.request.schema.json"),
		old:  `"lease_id"`,
		new:  `"x91"`,
	})

	after, err := schemaSetDigest(root)
	if err != nil {
		t.Fatal(err)
	}

	if after == before {
		t.Fatal("schema set digest did not change after referenced schema drift")
	}
}

func schemaSetDigest(root string) (string, error) {
	paths, err := filesUnder(root)
	if err != nil {
		return "", err
	}
	slices.Sort(paths)
	hash := sha256.New()
	for _, path := range paths {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return "", err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		hash.Write([]byte(filepath.ToSlash(relative)))
		hash.Write([]byte{0})
		hash.Write(content)
		hash.Write([]byte{0})
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}
