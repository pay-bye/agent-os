package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

type helpCase struct {
	name string
	args []string
	want []string
}

func TestTopLevelAndServeHelpColdDoorHaveNoSideEffects(t *testing.T) {
	requireHelpCases(t, []helpCase{
		{
			name: "top level long",
			args: []string{"--help"},
			want: []string{"commands:", "sits quiet until commanded", "operator key"},
		},
		{
			name: "top level short",
			args: []string{"-h"},
			want: []string{"commands:", "serve", "credential generate", "preview"},
		},
		{
			name: "serve",
			args: []string{"serve", "--help"},
			want: []string{
				"--operator-verifier-file",
				"sha256-base64url",
				"Operator-Key",
				"/operations/instructions/pause",
				"/operations/instructions/release-expired-lease",
				"/operations/instructions/force-release-lease",
				"/operations/instructions/move-item",
				"/operations/instructions/move-entries",
				"/operations/instructions/move-available",
				"/operations/instructions/drop",
				"/operations/instructions/route-outstanding",
			},
		},
	})
}

func TestCommandHelpColdDoorHasNoSideEffects(t *testing.T) {
	requireHelpCases(t, []helpCase{
		{
			name: "credential group",
			args: []string{"credential", "--help"},
			want: []string{"credential generate", "stores nothing", "rotates nothing"},
		},
		{
			name: "credential generate",
			args: []string{"credential", "generate", "--help"},
			want: []string{"prints one raw key", "verifier_digest", "bearer authority"},
		},
		{
			name: "init",
			args: []string{"init", "-h"},
			want: []string{"init", "--from", "writes", "does not read the database"},
		},
		{
			name: "preview",
			args: []string{"preview", "--help"},
			want: []string{"preview", "reads", "mutates no Registry vocabulary"},
		},
		{
			name: "apply",
			args: []string{"apply", "-h"},
			want: []string{"apply", "reads", "mutates Registry vocabulary"},
		},
	})
}

func requireHelpCases(t *testing.T, tests []helpCase) {
	t.Helper()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			calls := recordingCalls{}

			code := run(context.Background(), test.args, &output, &output, &calls)

			if code != 0 {
				t.Fatalf("exit = %d, output=%s", code, output.String())
			}
			if calls.any() {
				t.Fatalf("unexpected call: %+v", calls)
			}
			for _, want := range test.want {
				requireContains(t, output.String(), want)
			}
			requireHelpOmitsRuntimeValues(t, output.String())
		})
	}
}

func TestHelpWithConfiguredLookingValuesDoesNotPrintThem(t *testing.T) {
	var output bytes.Buffer
	calls := recordingCalls{}

	code := run(context.Background(), []string{
		"serve",
		"--database-url", "postgres://user:secret@host/db",
		"--verifier-digest", digest(t),
		"--operator-verifier-file", "/tmp/not-read",
		"--help",
	}, &output, &output, &calls)

	if code != 0 {
		t.Fatalf("exit = %d, output=%s", code, output.String())
	}
	if calls.any() {
		t.Fatalf("unexpected call: %+v", calls)
	}
	requireHelpOmitsRuntimeValues(t, output.String(), "secret", digest(t), "/tmp/not-read")
}

func TestServeLoadsOperatorVerifierFile(t *testing.T) {
	var output bytes.Buffer
	path := writeCommandVerifierFile(t, `{"algorithm":"sha256-base64url","digest":"`+digest(t)+`"}`)
	calls := recordingCalls{}

	code := run(context.Background(), []string{
		"serve",
		"--database-url", "postgres://u:p@host/db",
		"--listen", "127.0.0.1:0",
		"--verifier-digest", digest(t),
		"--operator-verifier-file", path,
	}, &output, &output, &calls)

	if code != 0 {
		t.Fatalf("exit = %d, output=%s", code, output.String())
	}
	if calls.last != "serve" {
		t.Fatalf("call = %q, want serve", calls.last)
	}
	if calls.serve.operator == nil || !calls.serve.operator.Accepts("") {
		t.Fatal("operator verifier was not passed to serve")
	}
	requireHelpOmitsRuntimeValues(t, output.String(), path)
}

func TestServeRejectsInvalidOperatorVerifierFileBeforeServing(t *testing.T) {
	var output bytes.Buffer
	digestValue := "bad-digest-sentinel"
	path := writeCommandVerifierFile(t, `{"algorithm":"sha256-base64url","digest":"`+digestValue+`"}`)
	calls := recordingCalls{}

	code := run(context.Background(), []string{
		"serve",
		"--database-url", "postgres://u:p@host/db",
		"--listen", "127.0.0.1:0",
		"--verifier-digest", digest(t),
		"--operator-verifier-file", path,
	}, &output, &output, &calls)

	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if calls.any() {
		t.Fatalf("unexpected call: %+v", calls)
	}
	requireContains(t, output.String(), "invalid_operator_verifier")
	requireHelpOmitsRuntimeValues(t, output.String(), path, digestValue)
}

func requireHelpOmitsRuntimeValues(t *testing.T, text string, values ...string) {
	t.Helper()

	for _, value := range values {
		if value != "" && strings.Contains(text, value) {
			t.Fatalf("help leaked %q in %s", value, text)
		}
	}
}
