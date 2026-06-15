package credential

import (
	"crypto/sha256"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOperatorVerifierAcceptsStrictDocument(t *testing.T) {
	path := writeOperatorVerifierFile(t, `{"algorithm":"sha256-base64url","digest":"`+filledDigest+`"}`, 0o600)

	verifier, err := LoadOperatorVerifier(OperatorVerifierInput{File: path})
	if err != nil {
		t.Fatal(err)
	}

	if !verifier.Accepts("filled") {
		t.Fatal("operator key rejected")
	}
}

func TestLoadOperatorVerifierRejectsInvalidDocument(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "missing algorithm", body: `{"digest":"` + filledDigest + `"}`},
		{name: "wrong algorithm", body: `{"algorithm":"sha512-base64url","digest":"` + filledDigest + `"}`},
		{name: "missing digest", body: `{"algorithm":"sha256-base64url"}`},
		{name: "unknown field", body: `{"algorithm":"sha256-base64url","digest":"` + filledDigest + `","sample":[]}`},
		{name: "duplicate field", body: `{"algorithm":"sha256-base64url","digest":"` + filledDigest + `","digest":"` + filledDigest + `"}`},
		{name: "multiple documents", body: `{"algorithm":"sha256-base64url","digest":"` + filledDigest + `"} {}`},
		{name: "array", body: `[]`},
		{name: "nested", body: `{"algorithm":"sha256-base64url","digest":{"sample":"` + filledDigest + `"}}`},
		{name: "invalid digest", body: `{"algorithm":"sha256-base64url","digest":"invalid"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := writeOperatorVerifierFile(t, test.body, 0o600)

			_, err := LoadOperatorVerifier(OperatorVerifierInput{File: path})

			requireError(t, err, "invalid_operator_verifier")
		})
	}
}

func TestLoadOperatorVerifierRejectsLooseMode(t *testing.T) {
	path := writeOperatorVerifierFile(t, `{"algorithm":"sha256-base64url","digest":"`+filledDigest+`"}`, 0o644)

	_, err := LoadOperatorVerifier(OperatorVerifierInput{File: path})

	requireError(t, err, "invalid_operator_verifier")
}

func TestLoadOperatorVerifierUsesReplacementDocument(t *testing.T) {
	firstKey := "alpha"
	secondKey := "bravo"
	path := writeOperatorVerifierFile(t, operatorVerifierDocumentFor(firstKey), 0o600)
	first, err := LoadOperatorVerifier(OperatorVerifierInput{File: path})
	if err != nil {
		t.Fatal(err)
	}

	writeOperatorVerifierPath(t, path, operatorVerifierDocumentFor(secondKey), 0o600)
	second, err := LoadOperatorVerifier(OperatorVerifierInput{File: path})
	if err != nil {
		t.Fatal(err)
	}

	if !first.Accepts(firstKey) || first.Accepts(secondKey) {
		t.Fatal("first verifier did not match first document")
	}
	if !second.Accepts(secondKey) || second.Accepts(firstKey) {
		t.Fatal("replacement verifier did not match replacement document")
	}
}

func writeOperatorVerifierFile(t *testing.T, body string, mode os.FileMode) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "value.json")
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeOperatorVerifierPath(t *testing.T, path string, body string, mode os.FileMode) {
	t.Helper()

	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func operatorVerifierDocumentFor(key string) string {
	return `{"algorithm":"sha256-base64url","digest":"` + digestFor(key) + `"}`
}

func digestFor(key string) string {
	sum := sha256.Sum256([]byte(key))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
