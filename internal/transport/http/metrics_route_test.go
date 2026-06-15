package http

import (
	"context"
	"errors"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/registry"
)

func TestMetricsReturnsBoundedTextWithoutCommandExecution(t *testing.T) {
	collector := metrics.New(
		metrics.WithClock(fixedMetricsClock{}),
		metrics.WithBuild(metrics.Build{Version: "v1.2.3", Revision: "a1b2c3"}),
	)
	commands := &recordingCommands{}

	response := serveWithOptions(t, commands, nethttp.MethodGet, "/metrics", "", WithMetrics(collector))

	requireCode(t, response, 200)
	requireTextContent(t, response)
	requireText(t, response, "# TYPE process_start_time_seconds gauge\n")
	requireText(t, response, `build_info{version="v1.2.3",revision="a1b2c3"} 1`+"\n")
	requireCalls(t, commands.calls)
}

func TestMetricsRequiresCredentialBeforeCollection(t *testing.T) {
	store := &recordingMetricsStore{}
	collector := metrics.New(metrics.WithStore(store))
	request := httptest.NewRequest(nethttp.MethodGet, "/metrics", nil)

	response := serveRawWithOptions(t, &recordingCommands{}, request, WithMetrics(collector))

	requireCode(t, response, 401)
	requireUnauthorizedBody(t, response)
	if store.called {
		t.Fatal("metrics storage collector was called before credential acceptance")
	}
}

func TestMetricsRejectsRequestBodiesBeforeCollection(t *testing.T) {
	store := &recordingMetricsStore{}
	collector := metrics.New(metrics.WithStore(store))
	request := httptest.NewRequest(nethttp.MethodGet, "/metrics", strings.NewReader("unexpected"))

	response := serveRequestWithOptions(t, &recordingCommands{}, request, WithMetrics(collector))

	requireCode(t, response, 400)
	requireBody(t, response, map[string]any{"error": "invalid_input"})
	if store.called {
		t.Fatal("metrics storage collector was called after invalid input")
	}
}

func TestMetricsRejectsUnsupportedMethods(t *testing.T) {
	response := serveWithOptions(
		t,
		&recordingCommands{},
		nethttp.MethodPost,
		"/metrics",
		"",
		WithMetrics(metrics.New()),
	)

	requireCode(t, response, 400)
	requireBody(t, response, map[string]any{"error": "invalid_input"})
}

func TestMetricsRecordsRequestAndAuthObservations(t *testing.T) {
	collector := metrics.New(metrics.WithClock(fixedMetricsClock{}))
	commands := &recordingCommands{}
	_ = serveWithOptions(t, commands, nethttp.MethodGet, "/health", "", WithMetrics(collector))
	request := httptest.NewRequest(nethttp.MethodGet, "/metrics", nil)
	_ = serveRawWithOptions(t, commands, request, WithMetrics(collector))

	response := serveWithOptions(t, commands, nethttp.MethodGet, "/metrics", "", WithMetrics(collector))

	requireText(t, response, `requests_total{operation="health",result="completed",protocol="http"} 1`+"\n")
	requireText(t, response, `auth_rejections_total{family="unauthorized",protocol="http"} 1`+"\n")
}

func TestMetricsRecordsRoutingErrorOutcomes(t *testing.T) {
	for _, test := range routingErrorCases() {
		t.Run(test.name, func(t *testing.T) {
			collector := metrics.New(metrics.WithClock(fixedMetricsClock{}))
			commands := &recordingCommands{}
			test.configure(commands)

			failed := serveWithOptions(t, commands, nethttp.MethodPost, test.path, test.body, WithMetrics(collector))
			scrape := serveWithOptions(t, commands, nethttp.MethodGet, "/metrics", "", WithMetrics(collector))

			requireCode(t, failed, test.code)
			requireBody(t, failed, map[string]any{"error": test.response})
			requireCalls(t, commands.calls, test.call)
			requireText(t, scrape, `routing_results_total{outcome="`+test.outcome+`"} 1`+"\n")
		})
	}
}

type routingErrorCase struct {
	name      string
	path      string
	body      string
	code      int
	response  string
	call      string
	outcome   string
	configure func(*recordingCommands)
}

func routingErrorCases() []routingErrorCase {
	return []routingErrorCase{
		{
			name:     "submit no route",
			path:     "/submit",
			body:     submitRouteBody(),
			code:     404,
			response: "no_route",
			call:     "submit",
			outcome:  "no_route",
			configure: func(commands *recordingCommands) {
				commands.submitErr = registry.ErrNoRoute
			},
		},
		{
			name:     "ack no route",
			path:     "/ack",
			body:     resolutionRouteBody(),
			code:     404,
			response: "no_route",
			call:     "ack",
			outcome:  "no_route",
			configure: func(commands *recordingCommands) {
				commands.ackErr = registry.ErrNoRoute
			},
		},
		{
			name:     "nack route failure",
			path:     "/nack",
			body:     nackRouteBody(),
			code:     409,
			response: "conflict",
			call:     "nack",
			outcome:  "failed",
			configure: func(commands *recordingCommands) {
				commands.nackErr = errors.New("route append failed")
			},
		},
	}
}

func submitRouteBody() string {
	return `{
		"work_item_id": "x08",
		"item_kind": "x08",
		"payload": {"value": "x75"},
		"declared_needs": [
			{"need_kind": "x12", "payload": {"value": "first"}}
		]
	}`
}

func resolutionRouteBody() string {
	return `{
		"lease_id": "x16",
		"lease_token": "x-token",
		"declared_needs": [
			{"need_kind": "x12", "payload": {"order": 1}}
		]
	}`
}

func nackRouteBody() string {
	return `{
		"lease_id": "x56",
		"lease_token": "x-token",
		"failure_payload": {"reason": "x46"},
		"declared_needs": [
			{"need_kind": "x12", "payload": {"order": 1}}
		]
	}`
}

type fixedMetricsClock struct{}

func (fixedMetricsClock) Now() time.Time {
	return time.Unix(1_779_000_000, 0)
}

type recordingMetricsStore struct {
	called bool
}

func (s *recordingMetricsStore) Read(context.Context, time.Time) (metrics.Storage, error) {
	s.called = true
	return metrics.Storage{}, nil
}

func requireTextContent(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	want := "text/plain; version=0.0.4; charset=utf-8"
	if got := response.Header().Get("Content-Type"); got != want {
		t.Fatalf("content type = %q, want %q", got, want)
	}
}

func requireText(t *testing.T, response *httptest.ResponseRecorder, value string) {
	t.Helper()

	if !strings.Contains(response.Body.String(), value) {
		t.Fatalf("body missing %q:\n%s", value, response.Body.String())
	}
}
