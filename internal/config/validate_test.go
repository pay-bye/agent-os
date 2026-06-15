package config

import (
	"strings"
	"testing"
)

func TestLoadRejectsUnknownConfigKeys(t *testing.T) {
	path := writeFile(t, `version: 1
database_url: postgres://user:secret@host/db
listen: 127.0.0.1:7000
extra: value
`)

	_, err := Load(Input{File: path})

	requireError(t, err, "unknown_field")
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("secret leaked in error: %v", err)
	}
}

func TestLoadRejectsDuplicateKeys(t *testing.T) {
	path := writeFile(t, `version: 1
version: 1
`)

	_, err := Load(Input{File: path})

	requireError(t, err, "duplicate_key")
}

func TestLoadRejectsMissingRequiredValues(t *testing.T) {
	_, err := Load(Input{RequireDatabase: true, Env: mapEnv{}})

	requireError(t, err, "missing_database_url")
}
