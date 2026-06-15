package config

import (
	"testing"
	"time"
)

func TestLoadUsesFlagFileEnvPrecedence(t *testing.T) {
	path := writeFile(t, `version: 1
database_url: postgres://file-user:file-secret@host/db
listen: 127.0.0.1:7000
declaration: custom.yaml
shutdown_grace_seconds: 4
`)

	config, err := Load(Input{
		File:        path,
		DatabaseURL: "postgres://flag-user:flag-secret@host/db",
		Listen:      "127.0.0.1:8000",
		Declaration: "flag.yaml",
		Grace:       2 * time.Second,
		Env:         mapEnv{"DATABASE_URL": "postgres://env-user:env-secret@host/db"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.DatabaseURL != "postgres://flag-user:flag-secret@host/db" {
		t.Fatalf("database URL = %q", config.DatabaseURL)
	}
	if config.Listen != "127.0.0.1:8000" {
		t.Fatalf("listen = %q", config.Listen)
	}
	if config.Declaration != "flag.yaml" {
		t.Fatalf("declaration = %q", config.Declaration)
	}
	if config.Grace != 2*time.Second {
		t.Fatalf("grace = %s", config.Grace)
	}
}

func TestLoadFallsBackToFileThenEnv(t *testing.T) {
	path := writeFile(t, `version: 1
listen: 127.0.0.1:7000
shutdown_grace_seconds: 4
`)

	config, err := Load(Input{
		File: path,
		Env:  mapEnv{"DATABASE_URL": "postgres://env-user:env-secret@host/db"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.DatabaseURL != "postgres://env-user:env-secret@host/db" {
		t.Fatalf("database URL = %q", config.DatabaseURL)
	}
	if config.Listen != "127.0.0.1:7000" {
		t.Fatalf("listen = %q", config.Listen)
	}
}
