package commands

import "testing"

import "github.com/pay-bye/agent-os/internal/channel"

func TestExtendParsesRequestedExpiry(t *testing.T) {
	lease := mustLease(t, "x16", instant(10))
	commands := &recordingCommands{extendResult: leaseResult(lease)}

	response := serve(t, commands, "POST", "/extend", `{
		"lease_id": "x16",
		"lease_token": "x-token",
		"requested_expires_at": "2026-05-18T12:10:00Z"
	}`)

	requireCode(t, response, 200)
	requireJSONContent(t, response)
	requireCalls(t, commands.calls, "extend")
	if commands.extend.Token != channel.Token("x-token") {
		t.Fatalf("token = %q, want x-token", commands.extend.Token)
	}
	if !commands.extend.ExpiresAt.Equal(instant(10)) {
		t.Fatalf("requested expiry = %s, want %s", commands.extend.ExpiresAt, instant(10))
	}
	requireBody(t, response, map[string]any{
		"lease_id":   "x16",
		"expires_at": "2026-05-18T12:10:00Z",
	})
}

func TestExtendRejectsMissingToken(t *testing.T) {
	commands := &recordingCommands{}

	response := serve(t, commands, "POST", "/extend", `{
		"lease_id": "x16",
		"requested_expires_at": "2026-05-18T12:10:00Z"
	}`)

	requireCode(t, response, 400)
	requireCalls(t, commands.calls)
	requireBody(t, response, map[string]any{"error": "invalid_input"})
}

func TestExtendRejectsInvalidDateTime(t *testing.T) {
	commands := &recordingCommands{}

	response := serve(t, commands, "POST", "/extend", `{
		"lease_id": "x16",
		"lease_token": "x-token",
		"requested_expires_at": "not-a-date"
	}`)

	requireCode(t, response, 400)
	requireCalls(t, commands.calls)
	requireBody(t, response, map[string]any{"error": "invalid_input"})
}
