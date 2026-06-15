package http

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/internal/transport/http/probes"
)

func TestHealthReturnsLiveWithoutCommandExecution(t *testing.T) {
	commands := &recordingCommands{}

	response := serve(t, commands, "GET", "/health", "")

	requireCode(t, response, 200)
	requireJSONContent(t, response)
	requireBody(t, response, map[string]any{"result": "live"})
	requireCalls(t, commands.calls)
}

func TestHealthDoesNotReadReadiness(t *testing.T) {
	commands := &recordingCommands{}
	called := false

	response := serveWithOptions(t, commands, "GET", "/health", "", WithReadiness(func(context.Context) probes.Readiness {
		called = true
		return probes.AllReady()
	}))

	requireCode(t, response, 200)
	if called {
		t.Fatal("health read readiness")
	}
	requireCalls(t, commands.calls)
}

func TestReadyzReturnsAllChecksReadyWithoutCommandExecution(t *testing.T) {
	commands := &recordingCommands{}

	response := serve(t, commands, "GET", "/readyz", "")

	requireCode(t, response, 200)
	requireJSONContent(t, response)
	requireBody(t, response, map[string]any{
		"result": "ready",
		"checks": readyBody(),
	})
	requireCalls(t, commands.calls)
}

func TestReadyzReturnsNotReadyWithoutRawDetail(t *testing.T) {
	commands := &recordingCommands{}
	checks := probes.AllReady()
	checks.Storage = probes.NotReady

	response := serveWithOptions(t, commands, "GET", "/readyz", "", WithReadiness(func(context.Context) probes.Readiness {
		return checks
	}))

	requireCode(t, response, 503)
	requireJSONContent(t, response)
	requireBody(t, response, map[string]any{
		"result": "not_ready",
		"checks": withReadyCheck("storage", "not_ready"),
	})
	requireAbsent(t, response.Body.String(), "postgres://", "database_url", "SQL", "secret")
	requireCalls(t, commands.calls)
}

func TestReadyzRejectsCredentialBeforeReadiness(t *testing.T) {
	commands := &recordingCommands{}
	called := false
	request := httptest.NewRequest("GET", "/readyz", nil)

	response := serveRawWithOptions(t, commands, request, WithReadiness(func(context.Context) probes.Readiness {
		called = true
		return probes.AllReady()
	}))

	requireCode(t, response, 401)
	requireUnauthorizedBody(t, response)
	if called {
		t.Fatal("readyz read readiness before credential acceptance")
	}
	requireCalls(t, commands.calls)
}

func TestProbeRoutesRejectRequestBodiesBeforeHandling(t *testing.T) {
	for _, path := range []string{"/health", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			commands := &recordingCommands{}
			called := false
			request := httptest.NewRequest(nethttp.MethodGet, path, strings.NewReader(`{"unexpected": true}`))

			response := serveRequestWithOptions(t, commands, request, WithReadiness(func(context.Context) probes.Readiness {
				called = true
				return probes.AllReady()
			}))

			requireCode(t, response, 400)
			requireBody(t, response, map[string]any{"error": "invalid_input"})
			if called {
				t.Fatal("probe body reached readiness handler")
			}
			requireCalls(t, commands.calls)
		})
	}
}

func TestProbeRoutesRejectWrongMethodWithoutCommandExecution(t *testing.T) {
	for _, path := range []string{"/health", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			commands := &recordingCommands{}

			response := serve(t, commands, "POST", path, "{}")

			requireCode(t, response, 400)
			requireBody(t, response, map[string]any{"error": "invalid_input"})
			requireCalls(t, commands.calls)
		})
	}
}

func readyBody() map[string]any {
	return map[string]any{
		"startup":     "ready",
		"storage":     "ready",
		"migrations":  "ready",
		"verifier":    "ready",
		"declaration": "ready",
		"handler":     "ready",
	}
}

func withReadyCheck(key string, value string) map[string]any {
	checks := readyBody()
	checks[key] = value
	return checks
}

func requireAbsent(t *testing.T, text string, values ...string) {
	t.Helper()

	for _, value := range values {
		if strings.Contains(text, value) {
			t.Fatalf("body exposed %q: %s", value, text)
		}
	}
}
