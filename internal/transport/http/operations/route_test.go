package operations

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

func TestRouteReturnsOperationsReport(t *testing.T) {
	response := serve(t, nethttp.MethodGet, "/operations", "", staticOperations{report: availableReport()})

	requireCode(t, response, nethttp.StatusOK)
	requireBody(t, response, map[string]any{
		"generated_at":   "2026-05-18T12:00:00Z",
		"result":         "complete",
		"window_seconds": float64(300),
		"views":          map[string]any{"queue": map[string]any{"depth": float64(1)}},
		"unavailable":    []any{},
	})
}

func TestRouteObservesBoundedRequest(t *testing.T) {
	collector := metrics.New()

	response := serveWithMetrics(
		t,
		collector,
		nethttp.MethodGet,
		"/operations",
		"",
		staticOperations{report: availableReport()},
	)

	requireCode(t, response, nethttp.StatusOK)
	requireMetric(t, collector, `requests_total{operation="operations",result="completed",protocol="http"} 1`)
}

func TestRootReadObservesUnavailableReportAsFailed(t *testing.T) {
	collector := metrics.New()

	response := serveWithMetrics(t, collector, nethttp.MethodGet, "/operations", "", staticOperations{})

	requireCode(t, response, nethttp.StatusServiceUnavailable)
	requireMetric(t, collector, `requests_total{operation="operations",result="failed",protocol="http"} 1`)
	rejectMetric(t, collector, `requests_total{operation="operations",result="completed",protocol="http"} 1`)
}

func TestReadRoutePassesBoundedQuery(t *testing.T) {
	operations := &recordingOperations{}

	response := serve(
		t,
		nethttp.MethodGet,
		"/operations/channels/x15/items?limit=3&older_than_seconds=60&lease_view=held",
		"",
		operations,
	)

	requireCode(t, response, nethttp.StatusOK)
	if operations.itemQuery.Channel != "x15" || operations.itemQuery.Limit != 3 {
		t.Fatalf("item query = %+v", operations.itemQuery)
	}
	if operations.itemQuery.OlderThanSeconds != 60 || operations.itemQuery.Lease != "held" {
		t.Fatalf("item query = %+v", operations.itemQuery)
	}
}

func TestReadRoutesRejectMalformedQueriesBeforeCallingOperations(t *testing.T) {
	operations := panicOperations{}

	response := serve(t, nethttp.MethodGet, "/operations/channels?limit=201", "", operations)

	requireCode(t, response, nethttp.StatusBadRequest)
}

func TestReadRoutesObserveMalformedQueriesAsRejected(t *testing.T) {
	collector := metrics.New()

	response := serveWithMetrics(
		t,
		collector,
		nethttp.MethodGet,
		"/operations/channels?limit=201",
		"",
		panicOperations{},
	)

	requireCode(t, response, nethttp.StatusBadRequest)
	requireMetric(t, collector, `requests_total{operation="operations",result="rejected",protocol="http"} 1`)
	rejectMetric(t, collector, `requests_total{operation="operations",result="failed",protocol="http"} 1`)
}

func TestReadRoutesObserveBoundedRequests(t *testing.T) {
	collector := metrics.New()

	response := serveWithMetrics(t, collector, nethttp.MethodGet, "/operations/channels", "", staticOperations{})

	requireCode(t, response, nethttp.StatusOK)
	requireMetric(t, collector, `requests_total{operation="operations",result="completed",protocol="http"} 1`)
}

func serve(
	t *testing.T,
	method string,
	path string,
	body string,
	operations Operations,
) *httptest.ResponseRecorder {
	t.Helper()

	return serveWithMetrics(t, metrics.New(), method, path, body, operations)
}

func serveWithMetrics(
	t *testing.T,
	collector *metrics.Collector,
	method string,
	path string,
	body string,
	operations Operations,
) *httptest.ResponseRecorder {
	t.Helper()

	mux := nethttp.NewServeMux()
	Register(mux, diagnostics.Settings{Metrics: collector}, operations)
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

type staticOperations struct {
	report report
}

func (s staticOperations) Response(context.Context) OperationsReport {
	return s.report
}

func (s staticOperations) Channels(context.Context, ChannelQuery) (any, error) {
	return map[string]any{"items": []any{}}, nil
}

func (s staticOperations) ChannelItems(context.Context, ChannelItemQuery) (any, error) {
	return map[string]any{"items": []any{}}, nil
}

func (s staticOperations) Item(context.Context, string) (any, error) {
	return map[string]any{"work_item_id": "x08"}, nil
}

func (s staticOperations) ItemJournal(context.Context, ItemJournalQuery) (any, error) {
	return map[string]any{"events": []any{}}, nil
}

func (s staticOperations) Nodes(context.Context, NodeQuery) (any, error) {
	return map[string]any{"nodes": []any{}}, nil
}

type recordingOperations struct {
	staticOperations
	itemQuery ChannelItemQuery
}

func (o *recordingOperations) ChannelItems(_ context.Context, query ChannelItemQuery) (any, error) {
	o.itemQuery = query
	return map[string]any{"items": []any{}}, nil
}

type panicOperations struct{}

func (panicOperations) Response(context.Context) OperationsReport {
	panic("operations called")
}

func (panicOperations) Channels(context.Context, ChannelQuery) (any, error) {
	panic("operations called")
}

func (panicOperations) ChannelItems(context.Context, ChannelItemQuery) (any, error) {
	panic("operations called")
}

func (panicOperations) Item(context.Context, string) (any, error) {
	panic("operations called")
}

func (panicOperations) ItemJournal(context.Context, ItemJournalQuery) (any, error) {
	panic("operations called")
}

func (panicOperations) Nodes(context.Context, NodeQuery) (any, error) {
	panic("operations called")
}

type report struct {
	GeneratedAt   string         `json:"generated_at"`
	Result        string         `json:"result"`
	WindowSeconds int            `json:"window_seconds"`
	Views         map[string]any `json:"views"`
	Unavailable   []string       `json:"unavailable"`
	available     bool
}

func (r report) Available() bool {
	return r.available
}

func availableReport() report {
	return report{
		GeneratedAt:   "2026-05-18T12:00:00Z",
		Result:        "complete",
		WindowSeconds: 300,
		Views:         map[string]any{"queue": map[string]any{"depth": 1}},
		Unavailable:   []string{},
		available:     true,
	}
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
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %v, want %v", got, want)
	}
}

func requireMetric(t *testing.T, collector *metrics.Collector, line string) {
	t.Helper()

	text := collector.Text(context.Background())
	if !strings.Contains(text, line+"\n") {
		t.Fatalf("metric line %q missing from:\n%s", line, text)
	}
}

func rejectMetric(t *testing.T, collector *metrics.Collector, line string) {
	t.Helper()

	text := collector.Text(context.Background())
	if strings.Contains(text, line+"\n") {
		t.Fatalf("metric line %q present in:\n%s", line, text)
	}
}
