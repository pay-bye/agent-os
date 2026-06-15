package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/internal/kernel"
)

func TestInstructionRoutesRequireOperatorKeyBeforeDecode(t *testing.T) {
	for _, test := range instructionRouteCases() {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(`{`))
			request.Header.Set("Content-Type", "application/json")

			response := serveRequestWithOptions(t, &recordingCommands{}, request, WithOperatorKey(mustVerifier(t, otherCredential)))

			requireCode(t, response, http.StatusUnauthorized)
			requireUnauthorizedBody(t, response)
		})
	}
}

func TestInstructionRoutesRequireBoundaryCredentialBeforeOperatorKey(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/operations/instructions/pause", strings.NewReader(instructionPauseBody()))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Operator-Key", otherCredential)

	response := serveRawWithOptions(t, &recordingCommands{}, request, WithOperatorKey(mustVerifier(t, otherCredential)))

	requireCode(t, response, http.StatusUnauthorized)
	requireUnauthorizedBody(t, response)
}

func TestInstructionRoutesDecodeAndDispatchCommands(t *testing.T) {
	for _, test := range instructionRouteCases() {
		t.Run(test.name, func(t *testing.T) {
			commands := &recordingCommands{instructionResult: instructionResult()}
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Operator-Key", otherCredential)

			response := serveRequestWithOptions(t, commands, request, WithOperatorKey(mustVerifier(t, otherCredential)))

			requireCode(t, response, http.StatusOK)
			requireJSONContent(t, response)
			requireCalls(t, commands.calls, test.call)
			requireBody(t, response, map[string]any{
				"instruction_id": "x70",
				"result":         "applied",
				"event_ids":      []any{"x80"},
				"affected_count": float64(1),
				"affected_ids":   []any{"x08"},
			})
		})
	}
}

func TestInstructionPreconditionFailureReturnsConflictResponse(t *testing.T) {
	commands := &recordingCommands{
		instructionResult: kernel.InstructionResult{
			ID:                 "x70",
			Result:             kernel.InstructionPreconditionFailed,
			EventIDs:           []string{"x80"},
			FailedPrecondition: "lease_expired",
		},
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/operations/instructions/release-expired-lease",
		strings.NewReader(instructionLeaseBody()),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Operator-Key", otherCredential)

	response := serveRequestWithOptions(t, commands, request, WithOperatorKey(mustVerifier(t, otherCredential)))

	requireCode(t, response, http.StatusConflict)
	requireBody(t, response, map[string]any{
		"instruction_id":      "x70",
		"result":              "precondition_failed",
		"event_ids":           []any{"x80"},
		"affected_count":      float64(0),
		"affected_ids":        []any{},
		"failed_precondition": "lease_expired",
	})
}

type instructionRouteCase struct {
	name string
	path string
	body string
	call string
}

func instructionRouteCases() []instructionRouteCase {
	return []instructionRouteCase{
		{name: "pause", path: "/operations/instructions/pause", body: instructionPauseBody(), call: "pause instruction"},
		{name: "release expired lease", path: "/operations/instructions/release-expired-lease", body: instructionLeaseBody(), call: "release expired lease instruction"},
		{name: "force release lease", path: "/operations/instructions/force-release-lease", body: instructionLeaseBody(), call: "force release lease instruction"},
		{name: "move item", path: "/operations/instructions/move-item", body: instructionMoveItemBody(), call: "move item instruction"},
		{name: "move entries", path: "/operations/instructions/move-entries", body: instructionMoveEntriesBody(), call: "move entries instruction"},
		{name: "move available", path: "/operations/instructions/move-available", body: instructionMoveAvailableBody(), call: "move available instruction"},
		{name: "drop", path: "/operations/instructions/drop", body: instructionItemsBody(), call: "drop instruction"},
		{name: "route outstanding", path: "/operations/instructions/route-outstanding", body: instructionItemsBody(), call: "route outstanding instruction"},
	}
}

func instructionResult() kernel.InstructionResult {
	return kernel.InstructionResult{
		ID:          "x70",
		Result:      kernel.InstructionApplied,
		EventIDs:    []string{"x80"},
		AffectedIDs: []string{"x08"},
	}
}

func instructionPauseBody() string {
	return `{"instruction_id":"x70","node_key":"x17"}`
}

func instructionLeaseBody() string {
	return `{"instruction_id":"x70","lease_id":"x16"}`
}

func instructionMoveItemBody() string {
	return `{"instruction_id":"x70","work_item_id":"x08","source_channel_key":"x15","target_channel_key":"x68"}`
}

func instructionMoveEntriesBody() string {
	return `{"instruction_id":"x70","source_channel_key":"x15","target_channel_key":"x68","entry_ids":["x31","x32"]}`
}

func instructionMoveAvailableBody() string {
	return `{"instruction_id":"x70","source_channel_key":"x15","target_channel_key":"x68","limit":2}`
}

func instructionItemsBody() string {
	return `{"instruction_id":"x70","work_item_ids":["x08","x09"]}`
}
