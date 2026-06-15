package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type mapEnv map[string]string

func (m mapEnv) LookupEnv(key string) (string, bool) {
	value, ok := m[key]
	return value, ok
}

func writeFile(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
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
