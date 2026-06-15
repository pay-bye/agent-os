package credential

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadVerifierUsesOnlyFlagsAndFile(t *testing.T) {
	path := writeVerifierFile(t, `{"digests":["`+emptyDigest+`"]}`)

	verifier, err := LoadVerifier(VerifierInput{
		Digests: []string{filledDigest},
		File:    path,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !verifier.Accepts("filled") || !verifier.Accepts("") {
		t.Fatal("expected verifier to accept flag and file material")
	}
}

func TestLoadVerifierRejectsMissingMaterial(t *testing.T) {
	_, err := LoadVerifier(VerifierInput{})

	requireError(t, err, "missing_verifier")
}

func writeVerifierFile(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "verifier.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
