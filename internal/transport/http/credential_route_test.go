package http

import (
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/internal/transport/http/security/credential"
)

func TestRequiresConfiguredVerifier(t *testing.T) {
	_, err := New(&recordingCommands{}, credential.Verifier{})

	if err == nil {
		t.Fatal("expected empty verifier rejection")
	}
}

func TestRejectsMissingAndInvalidCredentialsOnEveryRoute(t *testing.T) {
	for _, route := range acceptedRoutes() {
		t.Run(route.name+" missing", func(t *testing.T) {
			commands := &recordingCommands{}
			request := route.request()

			response := serveRaw(t, commands, request)

			requireCode(t, response, 401)
			requireUnauthorizedBody(t, response)
			requireCalls(t, commands.calls)
		})

		t.Run(route.name+" invalid", func(t *testing.T) {
			commands := &recordingCommands{}
			request := route.request()
			setCredential(request, otherCredential)

			response := serveRaw(t, commands, request)

			requireCode(t, response, 401)
			requireUnauthorizedBody(t, response)
			requireCalls(t, commands.calls)
		})
	}
}

func TestRejectsMalformedCredentialHeaderOnEveryRoute(t *testing.T) {
	headers := []string{
		"Basic " + validCredential,
		"Bearer",
		"Bearer ",
		"bearer " + validCredential,
		"Bearer " + validCredential + " extra",
	}

	for _, route := range acceptedRoutes() {
		for _, header := range headers {
			t.Run(route.name+" "+header, func(t *testing.T) {
				commands := &recordingCommands{}
				request := route.request()
				request.Header.Set("Authorization", header)

				response := serveRaw(t, commands, request)

				requireCode(t, response, 401)
				requireUnauthorizedBody(t, response)
				requireCalls(t, commands.calls)
			})
		}
	}
}

func TestRejectsBeforeBodyDecode(t *testing.T) {
	commands := &recordingCommands{}
	request := httptest.NewRequest("POST", "/submit", strings.NewReader(`{`))
	request.Header.Set("Content-Type", "application/json")

	response := serveRaw(t, commands, request)

	requireCode(t, response, 401)
	requireUnauthorizedBody(t, response)
	requireCalls(t, commands.calls)
}

type routeCase struct {
	name    string
	method  string
	path    string
	body    string
	content string
}

func acceptedRoutes() []routeCase {
	return []routeCase{
		{name: "submit", method: "POST", path: "/submit", body: `{`, content: "application/json"},
		{name: "claim", method: "POST", path: "/claim", body: `{`, content: "application/json"},
		{name: "ack", method: "POST", path: "/ack", body: `{`, content: "application/json"},
		{name: "nack", method: "POST", path: "/nack", body: `{`, content: "application/json"},
		{name: "extend", method: "POST", path: "/extend", body: `{`, content: "application/json"},
		{name: "heartbeat", method: "POST", path: "/heartbeat", body: `{`, content: "application/json"},
		{name: "health", method: "GET", path: "/health"},
		{name: "readyz", method: "GET", path: "/readyz"},
		{name: "metrics", method: "GET", path: "/metrics"},
		{name: "operations", method: "GET", path: "/operations"},
		{name: "operation channels", method: "GET", path: "/operations/channels"},
		{name: "operation channel items", method: "GET", path: "/operations/channels/x15/items"},
		{name: "operation item", method: "GET", path: "/operations/items/x08"},
		{name: "operation item journal", method: "GET", path: "/operations/items/x08/journal"},
		{name: "operation nodes", method: "GET", path: "/operations/nodes"},
		{name: "compatibility", method: "GET", path: "/compatibility"},
	}
}

func (c routeCase) request() *nethttp.Request {
	request := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
	if c.content != "" {
		request.Header.Set("Content-Type", c.content)
	}
	return request
}
