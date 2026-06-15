//go:build integration

package http_test

import (
	"context"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/internal/processlog"
)

func TestSubmitEmitsBoundedProcessRecords(t *testing.T) {
	ctx := context.Background()
	recorder := &recordingRecorder{}
	routes := handlerWithRecorder(t, ctx, recorder, "x25", "x27", "x30", "x32")

	response := submitItem(t, routes)

	requireCode(t, response, 200)
	requireOperations(t, recorder.records, processlog.HTTPAccept, processlog.KernelCommandOperation, processlog.HTTPComplete)
	requireSameCorrelation(t, recorder.records)
	if recorder.records[1].CommandFamily != processlog.Submit {
		t.Fatalf("command family = %q, want submit", recorder.records[1].CommandFamily)
	}
	requireRecordsExclude(t, recorder.records, "x08", "x75", "x76", bearerCredential)
}

func TestCompatibilityEmitsBoundedProcessRecords(t *testing.T) {
	ctx := context.Background()
	recorder := &recordingRecorder{}
	routes := handlerWithRecorder(t, ctx, recorder)

	response := request(t, routes, nethttp.MethodGet, "/compatibility")

	requireCode(t, response, nethttp.StatusOK)
	requireOperations(t, recorder.records, processlog.HTTPAccept, processlog.HTTPComplete)
	requireSameCorrelation(t, recorder.records)
	requireNoCommandFamily(t, recorder.records)
}

func TestCompatibilityRejectionEmitsBoundedProcessRecords(t *testing.T) {
	ctx := context.Background()
	recorder := &recordingRecorder{}
	routes := handlerWithRecorder(t, ctx, recorder)

	response := request(t, routes, nethttp.MethodPost, "/compatibility")

	requireCode(t, response, nethttp.StatusBadRequest)
	requireOperations(t, recorder.records, processlog.HTTPAccept, processlog.HTTPReject)
	requireSameCorrelation(t, recorder.records)
	requireNoCommandFamily(t, recorder.records)
	if recorder.records[1].ErrorCode != processlog.InvalidInput {
		t.Fatalf("error code = %q, want %q", recorder.records[1].ErrorCode, processlog.InvalidInput)
	}
}

type recordingRecorder struct {
	records []processlog.Record
}

func (r *recordingRecorder) Record(record processlog.Record) {
	r.records = append(r.records, record)
}

func requireOperations(t *testing.T, records []processlog.Record, operations ...processlog.Operation) {
	t.Helper()

	if len(records) != len(operations) {
		t.Fatalf("records = %+v, want operations %v", records, operations)
	}
	for index, operation := range operations {
		if records[index].Operation != operation {
			t.Fatalf("record %d operation = %q, want %q", index, records[index].Operation, operation)
		}
	}
}

func requireSameCorrelation(t *testing.T, records []processlog.Record) {
	t.Helper()

	if len(records) == 0 {
		t.Fatal("expected records")
	}
	correlation := records[0].Correlation
	if !strings.HasPrefix(correlation, "p-") {
		t.Fatalf("correlation = %q, want generated process value", correlation)
	}
	for _, record := range records {
		if record.Correlation != correlation {
			t.Fatalf("correlation = %q, want %q; records=%+v", record.Correlation, correlation, records)
		}
	}
}

func request(t *testing.T, routes nethttp.Handler, method string, path string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, nil)
	request.Header.Set("Authorization", "Bearer "+bearerCredential)
	response := httptest.NewRecorder()
	routes.ServeHTTP(response, request)
	return response
}

func requireNoCommandFamily(t *testing.T, records []processlog.Record) {
	t.Helper()

	for index, record := range records {
		if record.CommandFamily != "" {
			t.Fatalf("record %d command family = %q, want empty", index, record.CommandFamily)
		}
	}
}

func requireRecordsExclude(t *testing.T, records []processlog.Record, forbidden ...string) {
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
