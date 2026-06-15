package credential

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	validCredential = "w8JQSjz8qsBqq4lUmX70-k3MsG6YHj5p_EyV_dLPyVk"
	otherCredential = "Po4o08mUIlP5_8pO0HbLR8rloXaHvd7s2u6H9d5aI1A"
)

func TestVerifierMatchesConfiguredDigest(t *testing.T) {
	verifier, err := NewVerifier(verifierDigest(otherCredential), verifierDigest(validCredential))
	if err != nil {
		t.Fatal(err)
	}

	if !verifier.Accepts(validCredential) {
		t.Fatal("credential rejected")
	}
	if verifier.Accepts("zKxCPHGQ1bqpFVUw6zrGd6l8b40bHfC2DXdrGp8L9Qg") {
		t.Fatal("unconfigured credential accepted")
	}
}

func TestNewVerifierRejectsInvalidDigests(t *testing.T) {
	tests := []struct {
		name    string
		digests []string
	}{
		{name: "empty"},
		{name: "blank", digests: []string{""}},
		{name: "not base64url", digests: []string{"***"}},
		{name: "padded", digests: []string{base64.RawURLEncoding.EncodeToString(make([]byte, 32)) + "="}},
		{name: "wrong length", digests: []string{base64.RawURLEncoding.EncodeToString(make([]byte, 31))}},
		{name: "duplicate", digests: []string{verifierDigest(validCredential), verifierDigest(validCredential)}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewVerifier(test.digests...)

			if err == nil {
				t.Fatal("expected digest rejection")
			}
		})
	}
}

func TestReadVerifierFileAcceptsStrictDigestObject(t *testing.T) {
	path := writeCredentialVerifierFile(t, `{"digests":["`+verifierDigest(validCredential)+`"]}`, 0o600)

	verifier, err := ReadVerifierFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if !verifier.Accepts(validCredential) {
		t.Fatal("credential rejected")
	}
}

func TestReadVerifierFileRejectsInvalidShape(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed", body: `{`},
		{name: "unknown key", body: `{"digests":["` + verifierDigest(validCredential) + `"],"extra":[]}`},
		{name: "empty list", body: `{"digests":[]}`},
		{name: "duplicate", body: `{"digests":["` + verifierDigest(validCredential) + `","` + verifierDigest(validCredential) + `"]}`},
		{name: "padded", body: `{"digests":["` + verifierDigest(validCredential) + `="]}`},
		{name: "non array", body: `{"digests":"` + verifierDigest(validCredential) + `"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := writeCredentialVerifierFile(t, test.body, 0o600)

			_, err := ReadVerifierFile(path)

			if err == nil {
				t.Fatal("expected verifier file rejection")
			}
		})
	}
}

func TestReadVerifierFileRejectsLoosePermissions(t *testing.T) {
	path := writeCredentialVerifierFile(t, `{"digests":["`+verifierDigest(validCredential)+`"]}`, 0o644)

	_, err := ReadVerifierFile(path)

	if err == nil {
		t.Fatal("expected verifier file mode rejection")
	}
}

func TestGenerateCredentialReturnsMatchingMaterial(t *testing.T) {
	first, err := GenerateCredential()
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateCredential()
	if err != nil {
		t.Fatal(err)
	}

	if first.Credential == "" || strings.Contains(first.Credential, "=") {
		t.Fatalf("credential = %q", first.Credential)
	}
	if first.VerifierDigest != verifierDigest(first.Credential) {
		t.Fatalf("digest = %q, want %q", first.VerifierDigest, verifierDigest(first.Credential))
	}
	if first.Credential == second.Credential {
		t.Fatal("generator returned duplicate credential")
	}
	requireGeneratedJSON(t, first)
}

func requireGeneratedJSON(t *testing.T, value GeneratedCredential) {
	t.Helper()

	content, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"credential":      value.Credential,
		"verifier_digest": value.VerifierDigest,
	}
	requireStringMap(t, got, want)
}

func requireStringMap(t *testing.T, got map[string]string, want map[string]string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("map = %v, want %v", got, want)
	}
	for key, wantValue := range want {
		if got[key] != wantValue {
			t.Fatalf("map[%s] = %q, want %q; map=%v", key, got[key], wantValue, got)
		}
	}
}

func writeCredentialVerifierFile(t *testing.T, body string, mode os.FileMode) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "q9")
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	return path
}
