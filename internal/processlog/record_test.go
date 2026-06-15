package processlog

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSinkWritesBoundedJSONLine(t *testing.T) {
	var output bytes.Buffer
	sink := NewSink(&output, fixedClock{})

	err := sink.Emit(KernelCommand(Submit, Succeeded, ""))

	if err != nil {
		t.Fatal(err)
	}
	record := decodeLine(t, output.String())
	requireFields(t, record, "command_family", "component", "correlation", "operation", "outcome", "protocol", "severity", "timestamp")
	requireValue(t, record, "timestamp", "2026-05-30T20:15:41Z")
	requireValue(t, record, "severity", "info")
	requireValue(t, record, "component", "kernel")
	requireValue(t, record, "operation", "kernel.command")
	requireValue(t, record, "outcome", "succeeded")
	requireValue(t, record, "command_family", "submit")
	requireValue(t, record, "protocol", "http")
}

func TestInvalidRecordCombinationsAreRejected(t *testing.T) {
	validCorrelation := Correlation()
	tests := []struct {
		name   string
		record Record
	}{
		{
			name: "wrong component for operation",
			record: Record{
				Severity:  Info,
				Component: HTTP,
				Operation: KernelCommandOperation,
				Outcome:   Succeeded,
			},
		},
		{
			name: "command family outside kernel command",
			record: Record{
				Severity:      Info,
				Component:     HTTP,
				Operation:     HTTPAccept,
				Outcome:       Started,
				Correlation:   validCorrelation,
				CommandFamily: Submit,
				Protocol:      HTTPProtocol,
			},
		},
		{
			name: "caller supplied correlation",
			record: Record{
				Severity:    Info,
				Component:   HTTP,
				Operation:   HTTPAccept,
				Outcome:     Started,
				Correlation: "x-caller-supplied",
				Protocol:    HTTPProtocol,
			},
		},
		{
			name: "unknown error code",
			record: Record{
				Severity:  Error,
				Component: Storage,
				Operation: StorageError,
				Outcome:   Failed,
				ErrorCode: Code("postgres://user:secret@host/db"),
			},
		},
		{
			name: "error code on success",
			record: Record{
				Severity:  Info,
				Component: Process,
				Operation: ProcessStart,
				Outcome:   Started,
				ErrorCode: InternalError,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			err := NewSink(&output, fixedClock{}).Emit(test.record)

			if err == nil {
				t.Fatal("expected invalid record to fail")
			}
			if output.Len() != 0 {
				t.Fatalf("output = %s, want none", output.String())
			}
		})
	}
}

func TestConstructorsEmitAcceptedEventClasses(t *testing.T) {
	correlation := Correlation()
	tests := []struct {
		name      string
		record    Record
		operation string
		outcome   string
		code      string
	}{
		{name: "process start", record: ProcessStarted(), operation: "process.start", outcome: "started"},
		{name: "process stop completed", record: ProcessStopped(Completed, ""), operation: "process.stop", outcome: "completed"},
		{name: "process stop failed", record: ProcessStopped(Failed, InternalError), operation: "process.stop", outcome: "failed", code: "internal.error"},
		{name: "config succeeded", record: ConfigValidated(Succeeded, ""), operation: "config.validate", outcome: "succeeded"},
		{name: "config failed", record: ConfigValidated(Failed, ConfigInvalid), operation: "config.validate", outcome: "failed", code: "config.invalid"},
		{name: "storage migrated", record: StorageMigrated(Succeeded, ""), operation: "storage.migrate", outcome: "succeeded"},
		{name: "storage migration failed", record: StorageMigrated(Failed, StorageMigration), operation: "storage.migrate", outcome: "failed", code: "storage.migration"},
		{name: "declaration preview", record: DeclarationPreviewed(Succeeded, ""), operation: "declaration.preview", outcome: "succeeded"},
		{name: "declaration preview failed", record: DeclarationPreviewed(Failed, DeclarationInvalid), operation: "declaration.preview", outcome: "failed", code: "declaration.invalid"},
		{name: "declaration apply", record: DeclarationApplied(Succeeded, ""), operation: "declaration.apply", outcome: "succeeded"},
		{name: "declaration apply failed", record: DeclarationApplied(Failed, DeclarationInvalid), operation: "declaration.apply", outcome: "failed", code: "declaration.invalid"},
		{name: "http accepted", record: HTTPAccepted(correlation), operation: "http.accept", outcome: "started"},
		{name: "http rejected", record: HTTPRejected(correlation, InvalidInput), operation: "http.reject", outcome: "rejected", code: "invalid.input"},
		{name: "http failed", record: HTTPFailed(correlation, InternalError), operation: "http.fail", outcome: "failed", code: "internal.error"},
		{name: "http completed", record: HTTPCompleted(correlation), operation: "http.complete", outcome: "completed"},
		{name: "auth rejected", record: AuthRejectedRecord(correlation), operation: "auth.reject", outcome: "rejected", code: "auth.rejected"},
		{name: "dependency failed", record: DependencyFailure(DependencyUnavailable), operation: "dependency.error", outcome: "failed", code: "dependency.unavailable"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer

			err := NewSink(&output, fixedClock{}).Emit(test.record)

			if err != nil {
				t.Fatal(err)
			}
			record := decodeLine(t, output.String())
			requireValue(t, record, "operation", test.operation)
			requireValue(t, record, "outcome", test.outcome)
			if test.code != "" {
				requireValue(t, record, "error_code", test.code)
			}
		})
	}
}

func TestConstructorsMapUnknownErrorCodesToInternalError(t *testing.T) {
	sentinel := "postgres://user:secret@host/db"
	var output bytes.Buffer

	err := NewSink(&output, fixedClock{}).Emit(StorageFailure(Code(sentinel)))

	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), sentinel) {
		t.Fatalf("record exposed sentinel: %s", output.String())
	}
	record := decodeLine(t, output.String())
	requireValue(t, record, "error_code", "internal.error")
}

func TestCorrelationIsGeneratedInsideProcess(t *testing.T) {
	first := Correlation()
	second := Correlation()

	if first == second {
		t.Fatal("correlations must be unique within the process")
	}
	if !strings.HasPrefix(first, "p-") || !strings.HasPrefix(second, "p-") {
		t.Fatalf("correlations = %q, %q, want process-local prefix", first, second)
	}
	if strings.Contains(first, "caller") || strings.Contains(second, "caller") {
		t.Fatalf("correlations include caller material: %q %q", first, second)
	}
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, 5, 30, 20, 15, 41, 0, time.UTC)
}

func decodeLine(t *testing.T, line string) map[string]string {
	t.Helper()

	if !strings.HasSuffix(line, "\n") {
		t.Fatalf("record = %q, want newline terminator", line)
	}
	var record map[string]string
	if err := json.Unmarshal([]byte(line), &record); err != nil {
		t.Fatal(err)
	}
	return record
}

func requireFields(t *testing.T, record map[string]string, fields ...string) {
	t.Helper()

	if len(record) != len(fields) {
		t.Fatalf("fields = %v, want %v", record, fields)
	}
	for _, field := range fields {
		if _, ok := record[field]; !ok {
			t.Fatalf("missing field %s in %v", field, record)
		}
	}
}

func requireValue(t *testing.T, record map[string]string, field string, want string) {
	t.Helper()

	if record[field] != want {
		t.Fatalf("%s = %q, want %q; record=%v", field, record[field], want, record)
	}
}
