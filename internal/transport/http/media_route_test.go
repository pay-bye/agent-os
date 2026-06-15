package http

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRejectsInvalidMethodBeforeCommandExecution(t *testing.T) {
	commands := &recordingCommands{}

	response := serve(t, commands, "GET", "/submit", `{}`)

	requireCode(t, response, 400)
	requireJSONContent(t, response)
	requireCalls(t, commands.calls)
	requireBody(t, response, map[string]any{"error": "invalid_input"})
}

func TestRejectsMissingMediaTypeBeforeCommandExecution(t *testing.T) {
	commands := &recordingCommands{}
	request := httptest.NewRequest("POST", "/submit", strings.NewReader(`{}`))
	response := httptest.NewRecorder()

	handler := newHandler(t, commands)
	setCredential(request, validCredential)
	handler.ServeHTTP(response, request)

	requireCode(t, response, 400)
	requireJSONContent(t, response)
	requireCalls(t, commands.calls)
	requireBody(t, response, map[string]any{"error": "invalid_input"})
}

func TestRejectsNonJSONMediaTypeBeforeCommandExecution(t *testing.T) {
	commands := &recordingCommands{}
	request := httptest.NewRequest("POST", "/submit", strings.NewReader(`{
		"work_item_id": "x08",
		"item_kind": "x08",
		"payload": {},
		"declared_needs": []
	}`))
	request.Header.Set("Content-Type", "text/plain")

	response := serveRequest(t, commands, request)

	requireCode(t, response, 400)
	requireJSONContent(t, response)
	requireCalls(t, commands.calls)
	requireBody(t, response, map[string]any{"error": "invalid_input"})
}
