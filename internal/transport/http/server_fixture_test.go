package http

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/transport/http/probes"
	"github.com/pay-bye/agent-os/internal/transport/http/security/credential"
)

const (
	validCredential = "w8JQSjz8qsBqq4lUmX70-k3MsG6YHj5p_EyV_dLPyVk"
	otherCredential = "Po4o08mUIlP5_8pO0HbLR8rloXaHvd7s2u6H9d5aI1A"
)

func serve(
	t *testing.T,
	commands *recordingCommands,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	return serveWithOptions(t, commands, method, path, body)
}

func serveWithOptions(
	t *testing.T,
	commands *recordingCommands,
	method string,
	path string,
	body string,
	options ...Option,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return serveRequestWithOptions(t, commands, request, options...)
}

func serveRequest(
	t *testing.T,
	commands *recordingCommands,
	request *nethttp.Request,
) *httptest.ResponseRecorder {
	t.Helper()

	return serveRequestWithOptions(t, commands, request)
}

func serveRequestWithOptions(
	t *testing.T,
	commands *recordingCommands,
	request *nethttp.Request,
	options ...Option,
) *httptest.ResponseRecorder {
	t.Helper()

	setCredential(request, validCredential)
	return serveRawWithOptions(t, commands, request, options...)
}

func serveRaw(
	t *testing.T,
	commands *recordingCommands,
	request *nethttp.Request,
) *httptest.ResponseRecorder {
	t.Helper()

	return serveRawWithOptions(t, commands, request)
}

func serveRawWithOptions(
	t *testing.T,
	commands *recordingCommands,
	request *nethttp.Request,
	options ...Option,
) *httptest.ResponseRecorder {
	t.Helper()

	response := httptest.NewRecorder()
	newHandlerWithOptions(t, commands, options...).ServeHTTP(response, request)
	return response
}

func newHandler(t *testing.T, commands *recordingCommands) nethttp.Handler {
	t.Helper()

	return newHandlerWithOptions(t, commands)
}

func newHandlerWithOptions(t *testing.T, commands *recordingCommands, options ...Option) nethttp.Handler {
	t.Helper()

	verifier, err := credential.NewVerifier(verifierDigest(validCredential))
	if err != nil {
		t.Fatal(err)
	}
	options = append([]Option{WithReadiness(func(context.Context) probes.Readiness {
		return probes.AllReady()
	})}, options...)
	handler, err := New(commands, verifier, options...)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func setCredential(request *nethttp.Request, value string) {
	request.Header.Set("Authorization", "Bearer "+value)
}

func verifierDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func mustVerifier(t *testing.T, value string) credential.Verifier {
	t.Helper()

	verifier, err := credential.NewVerifier(verifierDigest(value))
	if err != nil {
		t.Fatal(err)
	}
	return verifier
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

func requireBody(t *testing.T, response *httptest.ResponseRecorder, want map[string]any) {
	t.Helper()

	var got map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("body = %v, want %v", got, want)
	}
	for key, wantValue := range want {
		if !reflect.DeepEqual(got[key], wantValue) {
			t.Fatalf("body[%s] = %v, want %v; body=%v", key, got[key], wantValue, got)
		}
	}
}

func requireUnauthorizedBody(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	if response.Body.String() != "Unauthorized\n" {
		t.Fatalf("body = %q, want generic unauthorized", response.Body.String())
	}
}

func requireCalls(t *testing.T, got []string, want ...string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("calls = %v, want %v", got, want)
	}
	for index, value := range want {
		if got[index] != value {
			t.Fatalf("calls = %v, want %v", got, want)
		}
	}
}

type recordingCommands struct {
	calls []string

	submit                         kernel.SubmitInput
	claim                          kernel.ClaimInput
	ack                            kernel.ResolutionInput
	nack                           kernel.ResolutionInput
	extend                         kernel.ExtendInput
	heartbeat                      kernel.HeartbeatInput
	pauseInstruction               kernel.PauseInstructionInput
	releaseExpiredLeaseInstruction kernel.LeaseInstructionInput
	forceReleaseLeaseInstruction   kernel.LeaseInstructionInput
	moveItemInstruction            kernel.MoveItemInstructionInput
	moveEntriesInstruction         kernel.MoveEntriesInstructionInput
	moveAvailableInstruction       kernel.MoveAvailableInstructionInput
	dropInstruction                kernel.ItemsInstructionInput
	routeOutstandingInstruction    kernel.ItemsInstructionInput

	submitResult      kernel.SubmitResult
	claimResult       kernel.ClaimResult
	ackResult         kernel.ResolutionResult
	nackResult        kernel.ResolutionResult
	extendResult      kernel.LeaseResult
	heartbeatResult   kernel.LeaseResult
	instructionResult kernel.InstructionResult

	submitErr      error
	claimErr       error
	ackErr         error
	nackErr        error
	extendErr      error
	heartbeatErr   error
	instructionErr error
}

func (c *recordingCommands) Submit(_ context.Context, input kernel.SubmitInput) (kernel.SubmitResult, error) {
	c.calls = append(c.calls, "submit")
	c.submit = input
	return c.submitResult, c.submitErr
}

func (c *recordingCommands) Claim(_ context.Context, input kernel.ClaimInput) (kernel.ClaimResult, error) {
	c.calls = append(c.calls, "claim")
	c.claim = input
	return c.claimResult, c.claimErr
}

func (c *recordingCommands) Ack(_ context.Context, input kernel.ResolutionInput) (kernel.ResolutionResult, error) {
	c.calls = append(c.calls, "ack")
	c.ack = input
	return c.ackResult, c.ackErr
}

func (c *recordingCommands) Nack(_ context.Context, input kernel.ResolutionInput) (kernel.ResolutionResult, error) {
	c.calls = append(c.calls, "nack")
	c.nack = input
	return c.nackResult, c.nackErr
}

func (c *recordingCommands) Extend(_ context.Context, input kernel.ExtendInput) (kernel.LeaseResult, error) {
	c.calls = append(c.calls, "extend")
	c.extend = input
	return c.extendResult, c.extendErr
}

func (c *recordingCommands) Heartbeat(_ context.Context, input kernel.HeartbeatInput) (kernel.LeaseResult, error) {
	c.calls = append(c.calls, "heartbeat")
	c.heartbeat = input
	return c.heartbeatResult, c.heartbeatErr
}

func (c *recordingCommands) PauseInstruction(
	_ context.Context,
	input kernel.PauseInstructionInput,
) (kernel.InstructionResult, error) {
	c.calls = append(c.calls, "pause instruction")
	c.pauseInstruction = input
	return c.instructionResult, c.instructionErr
}

func (c *recordingCommands) ReleaseExpiredLeaseInstruction(
	_ context.Context,
	input kernel.LeaseInstructionInput,
) (kernel.InstructionResult, error) {
	c.calls = append(c.calls, "release expired lease instruction")
	c.releaseExpiredLeaseInstruction = input
	return c.instructionResult, c.instructionErr
}

func (c *recordingCommands) ForceReleaseLeaseInstruction(
	_ context.Context,
	input kernel.LeaseInstructionInput,
) (kernel.InstructionResult, error) {
	c.calls = append(c.calls, "force release lease instruction")
	c.forceReleaseLeaseInstruction = input
	return c.instructionResult, c.instructionErr
}

func (c *recordingCommands) MoveItemInstruction(
	_ context.Context,
	input kernel.MoveItemInstructionInput,
) (kernel.InstructionResult, error) {
	c.calls = append(c.calls, "move item instruction")
	c.moveItemInstruction = input
	return c.instructionResult, c.instructionErr
}

func (c *recordingCommands) MoveEntriesInstruction(
	_ context.Context,
	input kernel.MoveEntriesInstructionInput,
) (kernel.InstructionResult, error) {
	c.calls = append(c.calls, "move entries instruction")
	c.moveEntriesInstruction = input
	return c.instructionResult, c.instructionErr
}

func (c *recordingCommands) MoveAvailableInstruction(
	_ context.Context,
	input kernel.MoveAvailableInstructionInput,
) (kernel.InstructionResult, error) {
	c.calls = append(c.calls, "move available instruction")
	c.moveAvailableInstruction = input
	return c.instructionResult, c.instructionErr
}

func (c *recordingCommands) DropInstruction(
	_ context.Context,
	input kernel.ItemsInstructionInput,
) (kernel.InstructionResult, error) {
	c.calls = append(c.calls, "drop instruction")
	c.dropInstruction = input
	return c.instructionResult, c.instructionErr
}

func (c *recordingCommands) RouteOutstandingInstruction(
	_ context.Context,
	input kernel.ItemsInstructionInput,
) (kernel.InstructionResult, error) {
	c.calls = append(c.calls, "route outstanding instruction")
	c.routeOutstandingInstruction = input
	return c.instructionResult, c.instructionErr
}
