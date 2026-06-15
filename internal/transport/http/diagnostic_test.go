package http

import (
	"encoding/json"
	"errors"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/processlog"
	"github.com/pay-bye/agent-os/internal/transport/http/security/credential"
)

func TestAuthRejectionRecordsBoundedDiagnostic(t *testing.T) {
	recorder := &recordingRecorder{}
	request := httptest.NewRequest(nethttp.MethodPost, "/submit", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer raw-credential-sentinel")
	request.Header.Set("X-Request-ID", "caller-correlation-sentinel")

	response := serveRawWithRecorder(t, &recordingCommands{}, request, recorder)

	requireCode(t, response, nethttp.StatusUnauthorized)
	requireUnauthorizedBody(t, response)
	requireRecords(t, recorder.records, processlog.AuthReject)
	record := recorder.records[0]
	if record.ErrorCode != processlog.AuthRejected {
		t.Fatalf("error code = %q, want %q", record.ErrorCode, processlog.AuthRejected)
	}
	requireGeneratedCorrelation(t, record.Correlation)
	requireNoRecordMaterial(t, recorder.records, "raw-credential-sentinel", "caller-correlation-sentinel")
}

func TestCommandDiagnosticsShareGeneratedCorrelationAndExcludePayloadValues(t *testing.T) {
	recorder := &recordingRecorder{}
	commands := &recordingCommands{submitResult: kernel.SubmitResult{Routed: true}}
	request := httptest.NewRequest(nethttp.MethodPost, "/submit", strings.NewReader(`{
		"work_item_id": "raw-work-sentinel",
		"item_kind": "kind-sentinel",
		"payload": {"secret": "payload-sentinel"},
		"declared_needs": []
	}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-ID", "caller-correlation-sentinel")
	setCredential(request, validCredential)

	response := serveRawWithRecorder(t, commands, request, recorder)

	requireCode(t, response, nethttp.StatusOK)
	requireRecords(t, recorder.records, processlog.HTTPAccept, processlog.KernelCommandOperation, processlog.HTTPComplete)
	correlation := recorder.records[0].Correlation
	requireGeneratedCorrelation(t, correlation)
	for _, record := range recorder.records {
		if record.Correlation != correlation {
			t.Fatalf("record correlation = %q, want %q; records=%+v", record.Correlation, correlation, recorder.records)
		}
	}
	if recorder.records[1].CommandFamily != processlog.Submit {
		t.Fatalf("command family = %q, want submit", recorder.records[1].CommandFamily)
	}
	requireNoRecordMaterial(
		t,
		recorder.records,
		"raw-work-sentinel",
		"kind-sentinel",
		"payload-sentinel",
		"caller-correlation-sentinel",
		validCredential,
	)
}

func TestRejectedRequestRecordsNoCommandFamily(t *testing.T) {
	recorder := &recordingRecorder{}
	request := httptest.NewRequest(nethttp.MethodPost, "/claim", strings.NewReader(`{"channel_key": ""}`))
	request.Header.Set("Content-Type", "application/json")
	setCredential(request, validCredential)

	response := serveRawWithRecorder(t, &recordingCommands{}, request, recorder)

	requireCode(t, response, nethttp.StatusBadRequest)
	requireRecords(t, recorder.records, processlog.HTTPAccept, processlog.HTTPReject)
	rejection := recorder.records[1]
	if rejection.CommandFamily != "" {
		t.Fatalf("command family = %q, want empty", rejection.CommandFamily)
	}
	if rejection.ErrorCode != processlog.InvalidInput {
		t.Fatalf("error code = %q, want %q", rejection.ErrorCode, processlog.InvalidInput)
	}
}

func TestCompatibilityDiagnosticsRecordSuccessWithoutCommandFamily(t *testing.T) {
	recorder := &recordingRecorder{}
	request := httptest.NewRequest(nethttp.MethodGet, "/compatibility", nil)
	setCredential(request, validCredential)

	response := serveRawWithRecorder(t, &recordingCommands{}, request, recorder)

	requireCode(t, response, nethttp.StatusOK)
	requireRecords(t, recorder.records, processlog.HTTPAccept, processlog.HTTPComplete)
	requireSameCorrelation(t, recorder.records)
	requireNoCommandFamily(t, recorder.records)
}

func TestCompatibilityDiagnosticsRecordWrongMethodRejection(t *testing.T) {
	recorder := &recordingRecorder{}
	request := httptest.NewRequest(nethttp.MethodPost, "/compatibility", nil)
	setCredential(request, validCredential)

	response := serveRawWithRecorder(t, &recordingCommands{}, request, recorder)

	requireCode(t, response, nethttp.StatusBadRequest)
	requireRecords(t, recorder.records, processlog.HTTPAccept, processlog.HTTPReject)
	requireSameCorrelation(t, recorder.records)
	requireNoCommandFamily(t, recorder.records)
	if recorder.records[1].ErrorCode != processlog.InvalidInput {
		t.Fatalf("error code = %q, want %q", recorder.records[1].ErrorCode, processlog.InvalidInput)
	}
}

func TestProbeDiagnosticsRecordWithoutCommandFamily(t *testing.T) {
	tests := []struct {
		path string
		code int
	}{
		{path: "/health", code: nethttp.StatusOK},
		{path: "/readyz", code: nethttp.StatusServiceUnavailable},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			recorder := &recordingRecorder{}
			request := httptest.NewRequest(nethttp.MethodGet, test.path, nil)
			setCredential(request, validCredential)

			response := serveRawWithRecorder(t, &recordingCommands{}, request, recorder)

			requireCode(t, response, test.code)
			requireRecords(t, recorder.records, processlog.HTTPAccept, processlog.HTTPComplete)
			requireSameCorrelation(t, recorder.records)
			requireNoCommandFamily(t, recorder.records)
		})
	}
}

func TestCommandFailureRecordsKernelAndRequestFailures(t *testing.T) {
	recorder := &recordingRecorder{}
	commands := &recordingCommands{claimErr: channel.ErrEmpty}

	response := serveWithRecorder(t, commands, nethttp.MethodPost, "/claim", `{
		"channel_key": "raw-channel-sentinel",
		"lease_id": "raw-lease-sentinel",
		"lease_seconds": 600
	}`, recorder)

	requireCode(t, response, nethttp.StatusNotFound)
	requireRecords(t, recorder.records, processlog.HTTPAccept, processlog.KernelCommandOperation, processlog.HTTPFail)
	kernelRecord := recorder.records[1]
	if kernelRecord.CommandFamily != processlog.Claim {
		t.Fatalf("command family = %q, want claim", kernelRecord.CommandFamily)
	}
	if kernelRecord.ErrorCode != processlog.EmptyQueue {
		t.Fatalf("kernel error code = %q, want %q", kernelRecord.ErrorCode, processlog.EmptyQueue)
	}
	if recorder.records[2].ErrorCode != processlog.EmptyQueue {
		t.Fatalf("request error code = %q, want %q", recorder.records[2].ErrorCode, processlog.EmptyQueue)
	}
	requireNoRecordMaterial(t, recorder.records, "raw-channel-sentinel", "raw-lease-sentinel")
}

func TestUnexpectedCommandFailureUsesInternalErrorCode(t *testing.T) {
	recorder := &recordingRecorder{}
	commands := &recordingCommands{heartbeatErr: errors.New("database-url-sentinel")}

	response := serveWithRecorder(t, commands, nethttp.MethodPost, "/heartbeat", `{
		"lease_id": "lease-sentinel",
		"lease_token": "token-sentinel"
	}`, recorder)

	requireCode(t, response, nethttp.StatusConflict)
	requireRecords(t, recorder.records, processlog.HTTPAccept, processlog.KernelCommandOperation, processlog.HTTPFail)
	if recorder.records[1].ErrorCode != processlog.InternalError {
		t.Fatalf("kernel error code = %q, want %q", recorder.records[1].ErrorCode, processlog.InternalError)
	}
	if recorder.records[2].ErrorCode != processlog.InternalError {
		t.Fatalf("request error code = %q, want %q", recorder.records[2].ErrorCode, processlog.InternalError)
	}
	requireNoRecordMaterial(t, recorder.records, "database-url-sentinel", "lease-sentinel", "token-sentinel")
}

type recordingRecorder struct {
	records []processlog.Record
}

func (r *recordingRecorder) Record(record processlog.Record) {
	r.records = append(r.records, record)
}

func serveWithRecorder(
	t *testing.T,
	commands *recordingCommands,
	method string,
	path string,
	body string,
	recorder *recordingRecorder,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	setCredential(request, validCredential)
	return serveRawWithRecorder(t, commands, request, recorder)
}

func serveRawWithRecorder(
	t *testing.T,
	commands *recordingCommands,
	request *nethttp.Request,
	recorder *recordingRecorder,
) *httptest.ResponseRecorder {
	t.Helper()

	verifier, err := credential.NewVerifier(verifierDigest(validCredential))
	if err != nil {
		t.Fatal(err)
	}
	handler, err := New(commands, verifier, WithRecorder(recorder))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func requireRecords(t *testing.T, records []processlog.Record, operations ...processlog.Operation) {
	t.Helper()

	if len(records) != len(operations) {
		t.Fatalf("records = %+v, want operations %v", records, operations)
	}
	for index, operation := range operations {
		if records[index].Operation != operation {
			t.Fatalf("record %d operation = %q, want %q; records=%+v", index, records[index].Operation, operation, records)
		}
	}
}

func requireGeneratedCorrelation(t *testing.T, value string) {
	t.Helper()

	if value == "" || !strings.HasPrefix(value, "p-") {
		t.Fatalf("correlation = %q, want generated process correlation", value)
	}
	if strings.Contains(value, "caller") || strings.Contains(value, "sentinel") {
		t.Fatalf("correlation includes caller material: %q", value)
	}
}

func requireSameCorrelation(t *testing.T, records []processlog.Record) {
	t.Helper()

	if len(records) == 0 {
		t.Fatal("expected records")
	}
	correlation := records[0].Correlation
	requireGeneratedCorrelation(t, correlation)
	for _, record := range records {
		if record.Correlation != correlation {
			t.Fatalf("correlation = %q, want %q; records=%+v", record.Correlation, correlation, records)
		}
	}
}

func requireNoCommandFamily(t *testing.T, records []processlog.Record) {
	t.Helper()

	for index, record := range records {
		if record.CommandFamily != "" {
			t.Fatalf("record %d command family = %q, want empty", index, record.CommandFamily)
		}
	}
}

func requireNoRecordMaterial(t *testing.T, records []processlog.Record, forbidden ...string) {
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
