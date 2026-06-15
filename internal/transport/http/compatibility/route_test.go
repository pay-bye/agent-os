package compatibility

import (
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
)

func TestRouteReturnsCompatibilityContract(t *testing.T) {
	response := serve(t, nethttp.MethodGet, "/compatibility")

	requireCode(t, response, nethttp.StatusOK)
	requireJSONContent(t, response)
	var body Compatibility
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.ContractVersion != "v1" || len(body.Routes) == 0 {
		t.Fatalf("contract = %+v", body)
	}
}

func TestRouteRejectsUnsupportedMethod(t *testing.T) {
	response := serve(t, nethttp.MethodPost, "/compatibility")

	requireCode(t, response, nethttp.StatusBadRequest)
}

func serve(t *testing.T, method string, path string) *httptest.ResponseRecorder {
	t.Helper()

	mux := nethttp.NewServeMux()
	Register(mux, diagnostics.Settings{Metrics: metrics.New()})
	request := httptest.NewRequest(method, path, nil)
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

func requireJSONContent(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q, want application/json", got)
	}
}
