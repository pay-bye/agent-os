package commands

import (
	"context"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func serve(
	t *testing.T,
	commands *recordingCommands,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux := nethttp.NewServeMux()
	Register(mux, commands, diagnostics.Settings{Metrics: metrics.New()})
	mux.ServeHTTP(response, request)
	return response
}

func requireNeedOrder(t *testing.T, got []workitem.DeclaredNeedInput, want ...string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("declared needs = %d, want %d", len(got), len(want))
	}
	for index, kind := range want {
		if got[index].Kind != registry.NeedKindKey(kind) {
			t.Fatalf("declared need %d = %q, want %q", index, got[index].Kind, kind)
		}
	}
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

func instant(minute int) time.Time {
	return time.Date(2026, 5, 18, 12, minute, 0, 0, time.UTC)
}

func leaseResult(lease channel.Lease) kernel.LeaseResult {
	return kernel.LeaseResult{Lease: lease}
}

func mustLease(t *testing.T, id string, expiresAt time.Time) channel.Lease {
	t.Helper()

	lease, err := channel.NewLease(channel.LeaseInput{
		ID:        channel.LeaseID(id),
		Entry:     channel.EntryID("x32"),
		Channel:   registry.ChannelKey("x15"),
		WorkItem:  workitem.ID("x08"),
		GrantedAt: instant(0),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	return lease
}

type recordingCommands struct {
	calls []string

	submit    kernel.SubmitInput
	claim     kernel.ClaimInput
	ack       kernel.ResolutionInput
	nack      kernel.ResolutionInput
	extend    kernel.ExtendInput
	heartbeat kernel.HeartbeatInput

	submitResult    kernel.SubmitResult
	claimResult     kernel.ClaimResult
	ackResult       kernel.ResolutionResult
	nackResult      kernel.ResolutionResult
	extendResult    kernel.LeaseResult
	heartbeatResult kernel.LeaseResult

	submitErr    error
	claimErr     error
	ackErr       error
	nackErr      error
	extendErr    error
	heartbeatErr error
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
