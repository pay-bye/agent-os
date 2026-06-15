package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/config"
	"github.com/pay-bye/agent-os/internal/declaration"
	"github.com/pay-bye/agent-os/internal/declaration/execution"
	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/processlog"
	"github.com/pay-bye/agent-os/internal/transport/http/security/credential"
)

func TestUnknownCommandFailsWithoutSideEffects(t *testing.T) {
	var output bytes.Buffer
	calls := recordingCalls{}

	code := run(context.Background(), []string{"mystery"}, &output, &output, &calls)

	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if calls.any() {
		t.Fatalf("unexpected call: %+v", calls)
	}
	requireContains(t, output.String(), "unknown command")
}

func TestRunReportsHelpWithoutCommand(t *testing.T) {
	var output bytes.Buffer
	calls := recordingCalls{}

	code := run(context.Background(), nil, &output, &output, &calls)

	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	requireContains(t, output.String(), "commands:")
}

func TestRuntimeConfigProvidesEnvironment(t *testing.T) {
	item := newRuntime(loadConfig([]string{"DATABASE_URL=postgres://x01"}))

	value, ok := item.LookupEnv("DATABASE_URL")

	if !ok || value != "postgres://x01" {
		t.Fatalf("DATABASE_URL = %q, %v", value, ok)
	}
}

func TestRuntimeGeneratesCredential(t *testing.T) {
	item := newRuntime(loadConfig(nil))

	credential, err := item.GenerateCredential()
	if err != nil {
		t.Fatal(err)
	}
	if credential.Credential == "" || credential.VerifierDigest == "" {
		t.Fatalf("credential = %+v", credential)
	}
}

func TestRuntimeInitializesDeclaration(t *testing.T) {
	item := newRuntime(loadConfig(nil))
	path := filepath.Join(t.TempDir(), "vocabulary.yaml")

	if err := item.Init(declaration.InitInput{Path: path}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeOperationsReportStorageFailure(t *testing.T) {
	item := newRuntime(loadConfig(nil))
	input := execution.Input{
		DatabaseURL: invalidDatabaseURL(),
		Storage:     item.Storage(),
	}

	if _, err := item.Preview(context.Background(), input); err == nil {
		t.Fatal("preview succeeded with invalid database")
	}
	if _, err := item.Apply(context.Background(), input); err == nil {
		t.Fatal("apply succeeded with invalid database")
	}
}

func TestRuntimeServeReportsStorageFailure(t *testing.T) {
	item := newRuntime(loadConfig(nil))

	err := item.Serve(context.Background(), serverInput{
		config: serverConfig(invalidDatabaseURL(), "127.0.0.1:0"),
	})

	if err == nil {
		t.Fatal("serve succeeded with invalid database")
	}
}

func TestNewHandlerReportsStorageReachabilityFailure(t *testing.T) {
	db, err := openDatabase(invalidDatabaseURL())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	generated, err := credential.GenerateCredential()
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := credential.NewVerifier(generated.VerifierDigest)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := newHandler(db, verifier, nil, nil, metrics.New(), declaration.Document{})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(nethttp.MethodGet, "/readyz", nil)
	request.Header.Set("Authorization", "Bearer "+generated.Credential)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != nethttp.StatusServiceUnavailable {
		t.Fatalf("code = %d, want 503; body=%s", response.Code, response.Body.String())
	}
	requireReadyzBody(t, response.Body.Bytes(), "not_ready")
	requireNotContains(t, response.Body.String(), "postgres://", "127.0.0.1", "user:pass")
}

func TestOpenMigratedDatabaseRecordsMigrationFailure(t *testing.T) {
	recorder := &commandRecorder{}

	_, err := openMigratedDatabase(context.Background(), invalidDatabaseURL(), recorder)

	if err == nil {
		t.Fatal("expected migration failure")
	}
	if len(recorder.records) != 1 {
		t.Fatalf("records = %+v, want one migration record", recorder.records)
	}
	record := recorder.records[0]
	if record.Operation != processlog.StorageMigrate {
		t.Fatalf("operation = %q, want %q", record.Operation, processlog.StorageMigrate)
	}
	if record.Outcome != processlog.Failed {
		t.Fatalf("outcome = %q, want failed", record.Outcome)
	}
	if record.ErrorCode != processlog.StorageMigration {
		t.Fatalf("error code = %q, want %q", record.ErrorCode, processlog.StorageMigration)
	}
	requireCommandRecordsExclude(t, recorder.records, "secret", "postgres://user:pass")
}

func TestServeStopsWhenContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := serve(ctx, serverConfig("", "127.0.0.1:0"), emptyHandler(), nil)

	if err != nil {
		t.Fatal(err)
	}
}

func TestServeRecordsProcessLifecycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	recorder := &commandRecorder{}

	err := serve(ctx, serverConfig("", "127.0.0.1:0"), emptyHandler(), recorder)

	if err != nil {
		t.Fatal(err)
	}
	if len(recorder.records) != 2 {
		t.Fatalf("records = %+v, want start and stop", recorder.records)
	}
	if recorder.records[0].Operation != processlog.ProcessStart {
		t.Fatalf("operation = %q, want %q", recorder.records[0].Operation, processlog.ProcessStart)
	}
	if recorder.records[1].Operation != processlog.ProcessStop {
		t.Fatalf("operation = %q, want %q", recorder.records[1].Operation, processlog.ProcessStop)
	}
	if recorder.records[1].Outcome != processlog.Completed {
		t.Fatalf("stop outcome = %q, want completed", recorder.records[1].Outcome)
	}
}

func TestServeRecordsStartBeforeAcceptingRequests(t *testing.T) {
	address := freeAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	recorder := newBlockingRecorder()
	done := make(chan error, 1)

	go func() {
		done <- serve(ctx, serverConfig("", address), emptyHandler(), recorder)
	}()
	recorder.requireStartCalled(t)

	requireNoResponse(t, "http://"+address, 300*time.Millisecond)

	recorder.releaseStart()
	cancel()
	requireServeDone(t, done)
}

func TestCommandDispatchesAcceptedVerbs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "serve",
			args: []string{
				"serve",
				"--database-url", "postgres://u:p@host/db",
				"--listen", "127.0.0.1:0",
				"--verifier-digest", digest(t),
			},
			want: "serve",
		},
		{name: "credential", args: []string{"credential", "generate"}, want: "credential"},
		{name: "init", args: []string{"init", "--from", filepath.Join(t.TempDir(), "vocabulary.yaml")}, want: "init"},
		{
			name: "preview",
			args: []string{"preview", "--database-url", "postgres://u:p@host/db", "--from", "vocabulary.yaml"},
			want: "preview",
		},
		{
			name: "apply",
			args: []string{"apply", "--database-url", "postgres://u:p@host/db", "--from", "vocabulary.yaml"},
			want: "apply",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			calls := recordingCalls{credential: credential.GeneratedCredential{
				Credential:     "opaque",
				VerifierDigest: digest(t),
			}}

			code := run(context.Background(), test.args, &output, &output, &calls)

			if code != 0 {
				t.Fatalf("exit = %d, output=%s", code, output.String())
			}
			if calls.last != test.want {
				t.Fatalf("call = %q, want %q", calls.last, test.want)
			}
		})
	}
}

func TestServeRecordsConfigDiagnostic(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	calls := recordingCalls{}

	code := run(
		context.Background(),
		[]string{
			"serve",
			"--database-url", "postgres://u:p@host/db",
			"--listen", "127.0.0.1:0",
			"--verifier-digest", digest(t),
		},
		&stdout,
		&stderr,
		&calls,
	)

	if code != 0 {
		t.Fatalf("exit = %d, stderr=%s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %s, want none", stdout.String())
	}
	records := decodeProcessRecords(t, stderr.String())
	requireProcessOperations(t, records, processlog.ConfigValidate)
}

func TestServeConfigFailureRecordsBoundedDiagnostic(t *testing.T) {
	var output bytes.Buffer

	code := run(
		context.Background(),
		[]string{"serve", "--listen", "127.0.0.1:0", "--verifier-digest", digest(t)},
		&output,
		&output,
		&recordingCalls{},
	)

	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	records := decodeProcessRecords(t, output.String())
	requireProcessOperations(t, records, processlog.ConfigValidate)
	if records[0].Outcome != string(processlog.Failed) {
		t.Fatalf("config outcome = %q, want failed", records[0].Outcome)
	}
	if records[0].ErrorCode != string(processlog.ConfigInvalid) {
		t.Fatalf("config error code = %q, want %q", records[0].ErrorCode, processlog.ConfigInvalid)
	}
}

func TestCredentialGeneratePrintsJSONAndWritesNoFiles(t *testing.T) {
	dir := t.TempDir()
	before := entries(t, dir)
	var output bytes.Buffer
	calls := recordingCalls{credential: credential.GeneratedCredential{
		Credential:     "raw",
		VerifierDigest: digest(t),
	}}

	code := run(context.Background(), []string{"credential", "generate"}, &output, &output, &calls)

	if code != 0 {
		t.Fatalf("exit = %d, output=%s", code, output.String())
	}
	var got map[string]string
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["credential"] != "raw" || got["verifier_digest"] != digest(t) {
		t.Fatalf("credential output = %v", got)
	}
	requireEntries(t, dir, before)
}

func TestPreviewAndApplyDoNotWriteDeltaAfterOperationFailure(t *testing.T) {
	for _, verb := range []string{"preview", "apply"} {
		t.Run(verb, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			calls := recordingCalls{err: errors.New("database_url contains <redacted>")}

			code := run(
				context.Background(),
				[]string{verb, "--database-url", "postgres://user:secret@host/db"},
				&stdout,
				&stderr,
				&calls,
			)

			if code == 0 {
				t.Fatal("expected non-zero exit")
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %s, want no delta", stdout.String())
			}
			requireContains(t, stderr.String(), "<redacted>")
		})
	}
}

func TestPreviewAndApplyReportOutputWriteFailure(t *testing.T) {
	for _, verb := range []string{"preview", "apply"} {
		t.Run(verb, func(t *testing.T) {
			var stderr bytes.Buffer
			calls := recordingCalls{}

			code := run(
				context.Background(),
				[]string{verb, "--database-url", "postgres://u:p@host/db"},
				errorWriter{},
				&stderr,
				&calls,
			)

			if code == 0 {
				t.Fatal("expected non-zero exit")
			}
			requireContains(t, stderr.String(), "write_failed")
		})
	}
}

func TestInitYesRequiresExplicitVocabularyInput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vocabulary.yaml")
	var output bytes.Buffer
	calls := recordingCalls{}

	code := run(context.Background(), []string{"init", "--yes", "--from", path}, &output, &output, &calls)

	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	requireContains(t, output.String(), "missing_explicit_vocabulary_input")
	if calls.last != "init" {
		t.Fatalf("call = %q, want init", calls.last)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("file exists after failed init: %v", err)
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write_failed")
}

func requireNotContains(t *testing.T, text string, forbidden ...string) {
	t.Helper()

	for _, value := range forbidden {
		if strings.Contains(text, value) {
			t.Fatalf("text exposed %q: %s", value, text)
		}
	}
}

func requireReadyzBody(t *testing.T, content []byte, storage string) {
	t.Helper()

	var body map[string]any
	if err := json.Unmarshal(content, &body); err != nil {
		t.Fatal(err)
	}
	checks, ok := body["checks"].(map[string]any)
	if !ok {
		t.Fatalf("checks = %v, want object", body["checks"])
	}
	if body["result"] != "not_ready" {
		t.Fatalf("result = %v, want not_ready", body["result"])
	}
	if checks["storage"] != storage {
		t.Fatalf("storage = %v, want %s; body=%v", checks["storage"], storage, body)
	}
}

func invalidDatabaseURL() string {
	return "postgres://user:pass@127.0.0.1:1/db?sslmode=disable&connect_timeout=1"
}

func serverConfig(databaseURL string, listen string) config.Values {
	return config.Values{
		DatabaseURL: databaseURL,
		Listen:      listen,
		Grace:       time.Second,
	}
}

func emptyHandler() nethttp.Handler {
	return nethttp.HandlerFunc(func(nethttp.ResponseWriter, *nethttp.Request) {})
}

func freeAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().String()
}

func requireNoResponse(t *testing.T, url string, duration time.Duration) {
	t.Helper()

	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		status, err := requestWithTimeout(url, 25*time.Millisecond)
		if err == nil {
			t.Fatalf("request completed before process.start emitted: %d", status)
		}
	}
}

func requestWithTimeout(url string, timeout time.Duration) (int, error) {
	client := nethttp.Client{Timeout: timeout}
	response, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	return response.StatusCode, nil
}

func requireServeDone(t *testing.T, done <-chan error) {
	t.Helper()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("serve did not stop")
	}
}

type processRecord struct {
	Operation string `json:"operation"`
	Outcome   string `json:"outcome"`
	ErrorCode string `json:"error_code"`
}

func decodeProcessRecords(t *testing.T, output string) []processRecord {
	t.Helper()

	var records []processRecord
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var record processRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return records
}

func requireProcessOperations(t *testing.T, records []processRecord, operations ...processlog.Operation) {
	t.Helper()

	if len(records) != len(operations) {
		t.Fatalf("records = %+v, want operations %v", records, operations)
	}
	for index, operation := range operations {
		if records[index].Operation != string(operation) {
			t.Fatalf("record %d operation = %q, want %q", index, records[index].Operation, operation)
		}
	}
}

type commandRecorder struct {
	records []processlog.Record
}

func (r *commandRecorder) Record(record processlog.Record) {
	r.records = append(r.records, record)
}

type blockingRecorder struct {
	started chan struct{}
	release chan struct{}
}

func newBlockingRecorder() *blockingRecorder {
	return &blockingRecorder{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (r *blockingRecorder) Record(record processlog.Record) {
	if record.Operation != processlog.ProcessStart {
		return
	}
	close(r.started)
	<-r.release
}

func (r *blockingRecorder) requireStartCalled(t *testing.T) {
	t.Helper()

	select {
	case <-r.started:
	case <-time.After(time.Second):
		t.Fatal("process.start was not recorded")
	}
}

func (r *blockingRecorder) releaseStart() {
	close(r.release)
}

func requireCommandRecordsExclude(t *testing.T, records []processlog.Record, forbidden ...string) {
	t.Helper()

	encoded, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, value := range forbidden {
		if strings.Contains(text, value) {
			t.Fatalf("records exposed %q: %s", value, text)
		}
	}
}
