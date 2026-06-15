package probes

import (
	"context"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
)

func TestHealthReturnsLiveWithoutReadiness(t *testing.T) {
	called := false

	response := serve(t, nethttp.MethodGet, "/health", "", func(context.Context) Readiness {
		called = true
		return AllReady()
	})

	requireCode(t, response, nethttp.StatusOK)
	requireBody(t, response, map[string]any{"result": "live"})
	if called {
		t.Fatal("health read readiness")
	}
}

func TestReadyzReportsNotReadyCheck(t *testing.T) {
	readiness := AllReady()
	readiness.Storage = NotReady

	response := serve(t, nethttp.MethodGet, "/readyz", "", func(context.Context) Readiness {
		return readiness
	})

	requireCode(t, response, nethttp.StatusServiceUnavailable)
	requireBody(t, response, map[string]any{
		"result": "not_ready",
		"checks": map[string]any{
			"startup":     "ready",
			"storage":     "not_ready",
			"migrations":  "ready",
			"verifier":    "ready",
			"declaration": "ready",
			"handler":     "ready",
		},
	})
}

func TestRoutesRejectBodiesBeforeReadiness(t *testing.T) {
	called := false

	response := serve(t, nethttp.MethodGet, "/readyz", "{}", func(context.Context) Readiness {
		called = true
		return AllReady()
	})

	requireCode(t, response, nethttp.StatusBadRequest)
	if called {
		t.Fatal("readiness called after invalid body")
	}
}

func serve(
	t *testing.T,
	method string,
	path string,
	body string,
	readiness ReadinessFunc,
) *httptest.ResponseRecorder {
	t.Helper()

	mux := nethttp.NewServeMux()
	Register(mux, diagnostics.Settings{Metrics: metrics.New()}, readiness)
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

func requireBody(t *testing.T, response *httptest.ResponseRecorder, want map[string]any) {
	t.Helper()

	var got map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	for key, value := range want {
		if !reflect.DeepEqual(got[key], value) {
			t.Fatalf("body[%s] = %v, want %v; body=%v", key, got[key], value, got)
		}
	}
}
