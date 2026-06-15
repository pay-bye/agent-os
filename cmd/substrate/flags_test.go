package main

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandReportsRedactedErrors(t *testing.T) {
	var output bytes.Buffer
	calls := recordingCalls{err: errors.New("database_url contains <redacted>")}

	code := run(
		context.Background(),
		[]string{"preview", "--database-url", "postgres://user:secret@host/db"},
		&output,
		&output,
		&calls,
	)

	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	requireContains(t, output.String(), "<redacted>")
	if strings.Contains(output.String(), "secret") {
		t.Fatalf("secret leaked in output: %s", output.String())
	}
}

func TestServeDoesNotReadOperatorVerifierFromEnvironment(t *testing.T) {
	var output bytes.Buffer
	calls := environmentCalls{
		values: map[string]string{
			"OPERATOR_VERIFIER_FILE": filepath.Join(t.TempDir(), "missing.json"),
		},
	}

	code := run(context.Background(), []string{
		"serve",
		"--database-url", "postgres://u:p@host/db",
		"--listen", "127.0.0.1:0",
		"--verifier-digest", digest(t),
	}, &output, &output, &calls)

	if code != 0 {
		t.Fatalf("exit = %d, output=%s", code, output.String())
	}
	if calls.last != "serve" {
		t.Fatalf("call = %q, want serve", calls.last)
	}
}

func TestServeRejectsOperatorVerifierConfigKey(t *testing.T) {
	var output bytes.Buffer
	path := writeCommandConfigFile(t, `version: 1
database_url: postgres://u:p@host/db
listen: 127.0.0.1:0
operator_verifier_file: /tmp/rejected.json
`)
	calls := recordingCalls{}

	code := run(context.Background(), []string{
		"serve",
		"--config", path,
		"--verifier-digest", digest(t),
	}, &output, &output, &calls)

	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if calls.any() {
		t.Fatalf("unexpected call: %+v", calls)
	}
	requireContains(t, output.String(), "unknown_field")
}

type environmentCalls struct {
	recordingCalls
	values map[string]string
}

func (c *environmentCalls) LookupEnv(key string) (string, bool) {
	value, ok := c.values[key]
	return value, ok
}
