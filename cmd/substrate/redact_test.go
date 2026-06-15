package main

import (
	"strings"
	"testing"
)

func TestRedactHidesDatabaseCredentialsAndVerifierMaterial(t *testing.T) {
	verifierDigest := digest(t)

	message := redact("postgres://user:secret@host/db failed with "+verifierDigest, []string{verifierDigest})

	if strings.Contains(message, "secret") || strings.Contains(message, verifierDigest) {
		t.Fatalf("redacted message leaked material: %s", message)
	}
}
