package commands

import (
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
)

func TestHeartbeatWritesLeaseResponse(t *testing.T) {
	lease := mustLease(t, "x16", instant(10))
	commands := &recordingCommands{heartbeatResult: kernel.LeaseResult{Lease: lease}}

	response := serve(t, commands, "POST", "/heartbeat", `{
		"lease_id": "x16",
		"lease_token": "x-token"
	}`)

	requireCode(t, response, 200)
	requireJSONContent(t, response)
	requireCalls(t, commands.calls, "heartbeat")
	if commands.heartbeat.Token != channel.Token("x-token") {
		t.Fatalf("token = %q, want x-token", commands.heartbeat.Token)
	}
	requireBody(t, response, map[string]any{
		"lease_id":   "x16",
		"expires_at": "2026-05-18T12:10:00Z",
	})
}

func TestHeartbeatRejectsMissingToken(t *testing.T) {
	commands := &recordingCommands{}

	response := serve(t, commands, "POST", "/heartbeat", `{"lease_id": "x16"}`)

	requireCode(t, response, 400)
	requireCalls(t, commands.calls)
	requireBody(t, response, map[string]any{"error": "invalid_input"})
}
