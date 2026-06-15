package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/internal/declaration"
	"github.com/pay-bye/agent-os/internal/declaration/execution"
	"github.com/pay-bye/agent-os/internal/transport/http/security/credential"
)

type recordingCalls struct {
	last       string
	credential credential.GeneratedCredential
	serve      serverInput
	err        error
}

func (c *recordingCalls) any() bool {
	return c.last != ""
}

func (c *recordingCalls) Serve(_ context.Context, input serverInput) error {
	c.last = "serve"
	c.serve = input
	return c.err
}

func (c *recordingCalls) GenerateCredential() (credential.GeneratedCredential, error) {
	c.last = "credential"
	if c.err != nil {
		return credential.GeneratedCredential{}, c.err
	}
	return c.credential, nil
}

func (c *recordingCalls) Init(input declaration.InitInput) error {
	c.last = "init"
	if c.err != nil {
		return c.err
	}
	return declaration.Init(input)
}

func (c *recordingCalls) Preview(context.Context, execution.Input) (declaration.Delta, error) {
	c.last = "preview"
	return declaration.Delta{Installable: true}, c.err
}

func (c *recordingCalls) Apply(context.Context, execution.Input) (declaration.Delta, error) {
	c.last = "apply"
	return declaration.Delta{Installable: true}, c.err
}

func digest(t *testing.T) string {
	t.Helper()
	return "47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU"
}

func entries(t *testing.T, dir string) []string {
	t.Helper()

	items, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name())
	}
	return names
}

func requireEntries(t *testing.T, dir string, want []string) {
	t.Helper()

	got := entries(t, dir)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("entries = %v, want %v", got, want)
	}
}

func requireContains(t *testing.T, text string, want string) {
	t.Helper()

	if !strings.Contains(text, want) {
		t.Fatalf("expected %q to contain %q", text, want)
	}
}

func writeCommandVerifierFile(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "value.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeCommandConfigFile(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
