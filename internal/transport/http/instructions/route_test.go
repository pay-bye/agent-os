package instructions

import (
	"context"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
)

func TestRouteRequiresOperatorKeyBeforeDecode(t *testing.T) {
	request := httptest.NewRequest(nethttp.MethodPost, "/operations/instructions/pause", strings.NewReader(`{`))
	request.Header.Set("Content-Type", "application/json")

	response := serveRaw(t, request, &recordingCommands{}, staticVerifier{accepted: false})

	requireCode(t, response, nethttp.StatusUnauthorized)
}

func TestRouteDecodesAndDispatchesInstruction(t *testing.T) {
	commands := &recordingCommands{result: appliedResult()}
	request := httptest.NewRequest(
		nethttp.MethodPost,
		"/operations/instructions/pause",
		strings.NewReader(`{"instruction_id":"x70","node_key":"x17"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Operator-Key", "x-key")

	response := serveRaw(t, request, commands, staticVerifier{accepted: true})

	requireCode(t, response, nethttp.StatusOK)
	if commands.pause.Node != "x17" {
		t.Fatalf("pause input = %+v", commands.pause)
	}
	requireBody(t, response, map[string]any{
		"instruction_id": "x70",
		"result":         "applied",
		"event_ids":      []any{"x80"},
		"affected_count": float64(1),
		"affected_ids":   []any{"x08"},
	})
}

func TestPreconditionFailureReturnsConflictResponse(t *testing.T) {
	commands := &recordingCommands{result: kernel.InstructionResult{
		ID:                 "x70",
		Result:             kernel.InstructionPreconditionFailed,
		FailedPrecondition: "lease_expired",
	}}
	request := httptest.NewRequest(
		nethttp.MethodPost,
		"/operations/instructions/release-expired-lease",
		strings.NewReader(`{"instruction_id":"x70","lease_id":"x16"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Operator-Key", "x-key")

	response := serveRaw(t, request, commands, staticVerifier{accepted: true})

	requireCode(t, response, nethttp.StatusConflict)
	requireBody(t, response, map[string]any{
		"instruction_id":      "x70",
		"result":              "precondition_failed",
		"event_ids":           []any{},
		"affected_count":      float64(0),
		"affected_ids":        []any{},
		"failed_precondition": "lease_expired",
	})
}

func serveRaw(
	t *testing.T,
	request *nethttp.Request,
	commands *recordingCommands,
	verifier staticVerifier,
) *httptest.ResponseRecorder {
	t.Helper()

	mux := nethttp.NewServeMux()
	Register(mux, diagnostics.Settings{Metrics: metrics.New()}, commands, verifier)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

type staticVerifier struct {
	accepted bool
}

func (v staticVerifier) Accepts(string) bool {
	return v.accepted
}

type recordingCommands struct {
	pause  kernel.PauseInstructionInput
	result kernel.InstructionResult
}

func (c *recordingCommands) PauseInstruction(
	_ context.Context,
	input kernel.PauseInstructionInput,
) (kernel.InstructionResult, error) {
	c.pause = input
	return c.result, nil
}

func (c *recordingCommands) ReleaseExpiredLeaseInstruction(
	context.Context,
	kernel.LeaseInstructionInput,
) (kernel.InstructionResult, error) {
	return c.result, nil
}

func (c *recordingCommands) ForceReleaseLeaseInstruction(
	context.Context,
	kernel.LeaseInstructionInput,
) (kernel.InstructionResult, error) {
	return c.result, nil
}

func (c *recordingCommands) MoveItemInstruction(
	context.Context,
	kernel.MoveItemInstructionInput,
) (kernel.InstructionResult, error) {
	return c.result, nil
}

func (c *recordingCommands) MoveEntriesInstruction(
	context.Context,
	kernel.MoveEntriesInstructionInput,
) (kernel.InstructionResult, error) {
	return c.result, nil
}

func (c *recordingCommands) MoveAvailableInstruction(
	context.Context,
	kernel.MoveAvailableInstructionInput,
) (kernel.InstructionResult, error) {
	return c.result, nil
}

func (c *recordingCommands) DropInstruction(
	context.Context,
	kernel.ItemsInstructionInput,
) (kernel.InstructionResult, error) {
	return c.result, nil
}

func (c *recordingCommands) RouteOutstandingInstruction(
	context.Context,
	kernel.ItemsInstructionInput,
) (kernel.InstructionResult, error) {
	return c.result, nil
}

func appliedResult() kernel.InstructionResult {
	return kernel.InstructionResult{
		ID:          "x70",
		Result:      kernel.InstructionApplied,
		EventIDs:    []string{"x80"},
		AffectedIDs: []string{"x08"},
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
