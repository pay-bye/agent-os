package scrape

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
)

func TestRouteReturnsMetricsText(t *testing.T) {
	response := serve(t, nethttp.MethodGet, "/metrics", "", metrics.New())

	requireCode(t, response, nethttp.StatusOK)
	if got := response.Header().Get("Content-Type"); got != "text/plain; version=0.0.4; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
	if !strings.Contains(response.Body.String(), "# TYPE process_start_time_seconds gauge\n") {
		t.Fatalf("metrics text missing process start:\n%s", response.Body.String())
	}
}

func TestRouteRejectsBodyBeforeCollecting(t *testing.T) {
	store := &recordingStore{}
	response := serve(t, nethttp.MethodGet, "/metrics", "unexpected", metrics.New(metrics.WithStore(store)))

	requireCode(t, response, nethttp.StatusBadRequest)
	if store.called {
		t.Fatal("store called after invalid request")
	}
}

func serve(
	t *testing.T,
	method string,
	path string,
	body string,
	collector *metrics.Collector,
) *httptest.ResponseRecorder {
	t.Helper()

	mux := nethttp.NewServeMux()
	Register(mux, diagnostics.Settings{Metrics: collector})
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func requireCode(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()

	if response.Code != want {
		t.Fatalf("code = %d, want %d, body=%s", response.Code, want, response.Body.String())
	}
}

type recordingStore struct {
	called bool
}

func (s *recordingStore) Read(context.Context, time.Time) (metrics.Storage, error) {
	s.called = true
	return metrics.Storage{}, nil
}
