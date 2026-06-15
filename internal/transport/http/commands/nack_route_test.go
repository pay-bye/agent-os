package commands

import (
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/registry"
)

func TestNackDispatchesFailurePayloadAndWritesRoutedResponse(t *testing.T) {
	commands := &recordingCommands{
		nackResult: kernel.ResolutionResult{
			Resolved: true,
			Routed:   true,
			Channel:  registry.ChannelKey("x68"),
		},
	}

	response := serve(t, commands, "POST", "/nack", `{
		"lease_id": "x56",
		"lease_token": "x-token",
		"failure_payload": {"reason": "x46"},
		"declared_needs": [
			{"need_kind": "x12", "target_node": "x17"}
		]
	}`)

	requireCode(t, response, 200)
	requireJSONContent(t, response)
	requireCalls(t, commands.calls, "nack")
	if commands.nack.Token != channel.Token("x-token") {
		t.Fatalf("token = %q, want x-token", commands.nack.Token)
	}
	if string(commands.nack.FailurePayload) != `{"reason":"x46"}` {
		t.Fatalf("failure payload = %s", commands.nack.FailurePayload)
	}
	if commands.nack.DeclaredNeeds[0].Target != registry.NodeKey("x17") {
		t.Fatalf("target = %q, want x17", commands.nack.DeclaredNeeds[0].Target)
	}
	requireBody(t, response, map[string]any{
		"resolved":    true,
		"routed":      true,
		"channel_key": "x68",
	})
}

func TestNackRejectsMissingToken(t *testing.T) {
	commands := &recordingCommands{}

	response := serve(t, commands, "POST", "/nack", `{
		"lease_id": "x56",
		"failure_payload": {"reason": "x46"},
		"declared_needs": []
	}`)

	requireCode(t, response, 400)
	requireCalls(t, commands.calls)
	requireBody(t, response, map[string]any{"error": "invalid_input"})
}
