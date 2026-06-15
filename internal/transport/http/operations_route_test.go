package http

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/internal/transport/http/operations"
)

func TestOperationsReturnsCompleteJSONWithoutCommandExecution(t *testing.T) {
	commands := &recordingCommands{}

	response := serveWithOptions(t, commands, nethttp.MethodGet, "/operations", "", WithOperations(completeOperations()))

	requireCode(t, response, nethttp.StatusOK)
	requireJSONContent(t, response)
	requireCalls(t, commands.calls)
	requireBody(t, response, map[string]any{
		"generated_at":   "2026-05-18T12:00:00Z",
		"result":         "complete",
		"window_seconds": float64(300),
		"unavailable":    []any{},
		"views":          completeViews(),
	})
}

func TestOperationsRequiresCredentialBeforeAssembly(t *testing.T) {
	request := httptest.NewRequest(nethttp.MethodGet, "/operations", nil)

	response := serveRawWithOptions(t, &recordingCommands{}, request, WithOperations(panicOperations{}))

	requireCode(t, response, nethttp.StatusUnauthorized)
	requireUnauthorizedBody(t, response)
}

func TestOperationsRejectsBodiesAndWriteMethodsBeforeAssembly(t *testing.T) {
	for _, test := range operationsInvalidRequestCases() {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "/operations", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")

			response := serveRequestWithOptions(t, &recordingCommands{}, request, WithOperations(panicOperations{}))

			requireCode(t, response, nethttp.StatusBadRequest)
			requireBody(t, response, map[string]any{"error": "invalid_input"})
		})
	}
}

func TestOperationsReturnsServiceUnavailableWhenAllGroupsFail(t *testing.T) {
	response := serveWithOptions(t, &recordingCommands{}, nethttp.MethodGet, "/operations", "", WithOperations(failingOperations()))

	requireCode(t, response, nethttp.StatusServiceUnavailable)
	requireJSONContent(t, response)
	requireBody(t, response, map[string]any{
		"generated_at":   "2026-05-18T12:00:00Z",
		"result":         "partial",
		"window_seconds": float64(300),
		"views":          map[string]any{},
		"unavailable": []any{
			"queue",
			"leases",
			"journal",
			"commands",
			"routing",
			"build",
			"compatibility",
		},
	})
}

func TestOperationReadRoutesReturnJSONWithoutCommandExecution(t *testing.T) {
	for _, test := range operationReadRouteCases() {
		t.Run(test.name, func(t *testing.T) {
			commands := &recordingCommands{}

			response := serveWithOptions(t, commands, nethttp.MethodGet, test.path, "", WithOperations(completeOperations()))

			requireCode(t, response, nethttp.StatusOK)
			requireJSONContent(t, response)
			requireCalls(t, commands.calls)
			requireBody(t, response, test.body)
		})
	}
}

func TestOperationReadRoutesRequireCredentialBeforeAssembly(t *testing.T) {
	request := httptest.NewRequest(nethttp.MethodGet, "/operations/channels", nil)

	response := serveRawWithOptions(t, &recordingCommands{}, request, WithOperations(panicOperations{}))

	requireCode(t, response, nethttp.StatusUnauthorized)
	requireUnauthorizedBody(t, response)
}

func TestOperationReadRoutesRejectBodiesAndMalformedQueriesBeforeAssembly(t *testing.T) {
	for _, test := range operationReadInvalidRequestCases() {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")

			response := serveRequestWithOptions(t, &recordingCommands{}, request, WithOperations(panicOperations{}))

			requireCode(t, response, nethttp.StatusBadRequest)
			requireBody(t, response, map[string]any{"error": "invalid_input"})
		})
	}
}

func TestOperationReadRoutesPassBoundedQueries(t *testing.T) {
	operations := &recordingOperations{}

	serveWithOptions(
		t,
		&recordingCommands{},
		nethttp.MethodGet,
		"/operations/channels/x15/items?limit=3&older_than_seconds=60&lease_view=held",
		"",
		WithOperations(operations),
	)

	if operations.itemQuery.Channel != "x15" {
		t.Fatalf("channel = %q, want x15", operations.itemQuery.Channel)
	}
	if operations.itemQuery.Limit != 3 || operations.itemQuery.OlderThanSeconds != 60 {
		t.Fatalf("query = %+v, want bounded item query", operations.itemQuery)
	}
	if operations.itemQuery.Lease != "held" {
		t.Fatalf("lease view = %q, want held", operations.itemQuery.Lease)
	}
}

type invalidRequestCase struct {
	name   string
	method string
	path   string
	body   string
}

func operationsInvalidRequestCases() []invalidRequestCase {
	return []invalidRequestCase{
		{name: "get body", method: nethttp.MethodGet, body: "{}"},
		{name: "post", method: nethttp.MethodPost},
		{name: "put", method: nethttp.MethodPut},
		{name: "patch", method: nethttp.MethodPatch},
		{name: "delete", method: nethttp.MethodDelete},
	}
}

func operationReadRouteCases() []struct {
	name string
	path string
	body map[string]any
} {
	return []struct {
		name string
		path string
		body map[string]any
	}{
		{name: "channels", path: "/operations/channels", body: channelListBody()},
		{name: "channel items", path: "/operations/channels/x15/items", body: channelItemsBody()},
		{name: "item detail", path: "/operations/items/x08", body: itemBody()},
		{name: "item journal", path: "/operations/items/x08/journal", body: journalBody()},
		{name: "nodes", path: "/operations/nodes", body: nodesBody()},
	}
}

func operationReadInvalidRequestCases() []invalidRequestCase {
	return []invalidRequestCase{
		{name: "channels body", method: nethttp.MethodGet, path: "/operations/channels", body: "{}"},
		{name: "channels write", method: nethttp.MethodPost, path: "/operations/channels"},
		{name: "channels bad limit", method: nethttp.MethodGet, path: "/operations/channels?limit=201"},
		{name: "channel items bad lease view", method: nethttp.MethodGet, path: "/operations/channels/x15/items?lease_view=token"},
		{name: "item detail query", method: nethttp.MethodGet, path: "/operations/items/x08?status=dead"},
		{name: "item journal bad index", method: nethttp.MethodGet, path: "/operations/items/x08/journal?after_append_index=-1"},
		{name: "nodes unknown query", method: nethttp.MethodGet, path: "/operations/nodes?best_node=true"},
	}
}

func completeOperations() operations.Operations {
	return staticOperations{report: completeReport()}
}

func failingOperations() operations.Operations {
	return staticOperations{report: failingReport()}
}

type staticOperations struct {
	report operationsReport
}

func (s staticOperations) Response(context.Context) operations.OperationsReport {
	return s.report
}

func (s staticOperations) Channels(context.Context, operations.ChannelQuery) (any, error) {
	return channelListBody(), nil
}

func (s staticOperations) ChannelItems(context.Context, operations.ChannelItemQuery) (any, error) {
	return channelItemsBody(), nil
}

func (s staticOperations) Item(context.Context, string) (any, error) {
	return itemBody(), nil
}

func (s staticOperations) ItemJournal(context.Context, operations.ItemJournalQuery) (any, error) {
	return journalBody(), nil
}

func (s staticOperations) Nodes(context.Context, operations.NodeQuery) (any, error) {
	return nodesBody(), nil
}

type panicOperations struct{}

func (panicOperations) Response(context.Context) operations.OperationsReport {
	panic("operations were called before request acceptance")
}

func (panicOperations) Channels(context.Context, operations.ChannelQuery) (any, error) {
	panic("operations were called before request acceptance")
}

func (panicOperations) ChannelItems(context.Context, operations.ChannelItemQuery) (any, error) {
	panic("operations were called before request acceptance")
}

func (panicOperations) Item(context.Context, string) (any, error) {
	panic("operations were called before request acceptance")
}

func (panicOperations) ItemJournal(context.Context, operations.ItemJournalQuery) (any, error) {
	panic("operations were called before request acceptance")
}

func (panicOperations) Nodes(context.Context, operations.NodeQuery) (any, error) {
	panic("operations were called before request acceptance")
}

type recordingOperations struct {
	staticOperations
	itemQuery operations.ChannelItemQuery
}

func (o *recordingOperations) ChannelItems(_ context.Context, query operations.ChannelItemQuery) (any, error) {
	o.itemQuery = query
	return channelItemsBody(), nil
}

type operationsReport struct {
	GeneratedAt   string         `json:"generated_at"`
	Result        string         `json:"result"`
	WindowSeconds int            `json:"window_seconds"`
	Views         map[string]any `json:"views"`
	Unavailable   []string       `json:"unavailable"`
	available     bool
}

func (r operationsReport) Available() bool {
	return r.available
}

func completeReport() operationsReport {
	return operationsReport{
		GeneratedAt:   "2026-05-18T12:00:00Z",
		Result:        "complete",
		WindowSeconds: 300,
		Unavailable:   []string{},
		Views:         completeViews(),
		available:     true,
	}
}

func failingReport() operationsReport {
	return operationsReport{
		GeneratedAt:   "2026-05-18T12:00:00Z",
		Result:        "partial",
		WindowSeconds: 300,
		Views:         map[string]any{},
		Unavailable: []string{
			"queue",
			"leases",
			"journal",
			"commands",
			"routing",
			"build",
			"compatibility",
		},
	}
}

func completeViews() map[string]any {
	return map[string]any{
		"queue": map[string]any{
			"channel_class":                "all",
			"depth":                        float64(12),
			"available":                    float64(7),
			"oldest_available_age_seconds": float64(84),
		},
		"leases": map[string]any{
			"channel_class": "all",
			"held":          float64(3),
			"expired":       float64(1),
		},
		"journal": map[string]any{
			"append_rate_per_second": float64(0.4),
			"window_seconds":         float64(300),
		},
		"commands": map[string]any{
			"succeeded": float64(40),
			"failed":    float64(2),
		},
		"routing": map[string]any{
			"routed":   float64(38),
			"unrouted": float64(4),
		},
		"build": map[string]any{
			"version":  "v1.2.3",
			"revision": "a1b2c3",
		},
		"compatibility": map[string]any{
			"contract_version": "v1",
			"features":         []any{"x91"},
			"routes": []any{
				map[string]any{"method": "GET", "path": "/x91"},
			},
		},
	}
}

func channelListBody() map[string]any {
	return map[string]any{
		"channels": []any{
			map[string]any{
				"channel_key":                  "x15",
				"node_key":                     "x17",
				"depth":                        float64(2),
				"available":                    float64(1),
				"oldest_available_age_seconds": float64(120),
			},
		},
	}
}

func channelItemsBody() map[string]any {
	return map[string]any{
		"items": []any{
			map[string]any{
				"entry_id":     "x01",
				"work_item_id": "x08",
				"channel_key":  "x15",
				"node_key":     "x17",
				"enqueued_at":  "2026-05-18T11:58:00Z",
				"available_at": "2026-05-18T11:59:00Z",
				"age_seconds":  float64(60),
				"lease": map[string]any{
					"lease_id":   "x13",
					"granted_at": "2026-05-18T12:00:00Z",
					"expires_at": "2026-05-18T12:01:00Z",
				},
			},
		},
	}
}

func itemBody() map[string]any {
	return map[string]any{
		"work_item_id": "x08",
		"item_kind":    "x03",
		"submitted_at": "2026-05-18T11:55:00Z",
		"channel_entry": map[string]any{
			"entry_id":     "x01",
			"channel_key":  "x15",
			"node_key":     "x17",
			"enqueued_at":  "2026-05-18T11:58:00Z",
			"available_at": "2026-05-18T11:59:00Z",
			"age_seconds":  float64(60),
		},
		"lease": map[string]any{
			"lease_id":    "x13",
			"channel_key": "x15",
			"granted_at":  "2026-05-18T12:00:00Z",
			"expires_at":  "2026-05-18T12:01:00Z",
		},
		"outstanding_need": map[string]any{
			"event_id":    "x21",
			"need_kind":   "x12",
			"target_node": "x17",
			"declared_at": "2026-05-18T11:59:30Z",
		},
	}
}

func journalBody() map[string]any {
	return map[string]any{
		"events": []any{
			map[string]any{
				"event_id":     "x21",
				"event_kind":   "x41",
				"appended_at":  "2026-05-18T12:00:00Z",
				"append_index": float64(1),
				"metadata": map[string]any{
					"work_item_id": "x08",
					"need_kind":    "x12",
				},
			},
		},
	}
}

func nodesBody() map[string]any {
	return map[string]any{
		"nodes": []any{
			map[string]any{
				"node_key":    "x17",
				"channel_key": "x15",
				"need_kinds":  []any{"x12"},
				"routable":    false,
			},
		},
	}
}
