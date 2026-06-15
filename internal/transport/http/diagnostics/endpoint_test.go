package diagnostics

import (
	"errors"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/processlog"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
)

func TestRequestEndpointRecordsAcceptedAndCompleted(t *testing.T) {
	recorder := &recordingRecorder{}
	handler := RequestEndpoint(metrics.Health, nethttp.MethodGet, Settings{
		Recorder: recorder,
		Metrics:  metrics.New(),
	}, func(*nethttp.Request) (codec.Response, error) {
		return codec.OK(map[string]string{"result": "ok"}), nil
	})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, httptest.NewRequest(nethttp.MethodGet, "/health", nil))

	requireCode(t, response, nethttp.StatusOK)
	requireOperations(t, recorder.records, processlog.HTTPAccept, processlog.HTTPComplete)
}

func TestRequestEndpointRejectsGuardFailures(t *testing.T) {
	recorder := &recordingRecorder{}
	handler := RequestEndpoint(metrics.Health, nethttp.MethodGet, Settings{
		Recorder: recorder,
		Metrics:  metrics.New(),
	}, func(*nethttp.Request) (codec.Response, error) {
		t.Fatal("call reached after guard failure")
		return codec.Response{}, nil
	}, RejectBody)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(nethttp.MethodGet, "/health", strings.NewReader("{}"))

	handler.ServeHTTP(response, request)

	requireCode(t, response, nethttp.StatusBadRequest)
	requireOperations(t, recorder.records, processlog.HTTPAccept, processlog.HTTPReject)
}

func TestRequestEndpointRecordsFailures(t *testing.T) {
	recorder := &recordingRecorder{}
	handler := RequestEndpoint(metrics.Health, nethttp.MethodGet, Settings{
		Recorder: recorder,
		Metrics:  metrics.New(),
	}, func(*nethttp.Request) (codec.Response, error) {
		return codec.Response{}, errors.New("storage unavailable")
	})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, httptest.NewRequest(nethttp.MethodGet, "/health", nil))

	requireCode(t, response, nethttp.StatusConflict)
	requireOperations(t, recorder.records, processlog.HTTPAccept, processlog.HTTPFail)
}

type recordingRecorder struct {
	records []processlog.Record
}

func (r *recordingRecorder) Record(record processlog.Record) {
	r.records = append(r.records, record)
}

func requireCode(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()

	if response.Code != want {
		t.Fatalf("code = %d, want %d, body=%s", response.Code, want, response.Body.String())
	}
}

func requireOperations(t *testing.T, records []processlog.Record, want ...processlog.Operation) {
	t.Helper()

	if len(records) != len(want) {
		t.Fatalf("records = %+v, want %v", records, want)
	}
	for index, operation := range want {
		if records[index].Operation != operation {
			t.Fatalf("record %d = %q, want %q; records=%+v", index, records[index].Operation, operation, records)
		}
	}
}
