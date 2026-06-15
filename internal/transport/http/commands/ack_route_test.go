package commands

import (
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/registry"
)

func TestAckDispatchesTokenAndPreservesNeedOrder(t *testing.T) {
	commands := &recordingCommands{
		ackResult: kernel.ResolutionResult{Resolved: true, Routed: false},
	}

	response := serve(t, commands, "POST", "/ack", `{
		"lease_id": "x16",
		"lease_token": "x-token",
		"declared_needs": [
			{"need_kind": "x12", "target_node": "x17", "payload": {"order": 1}},
			{"need_kind": "x13", "payload": {"order": 2}}
		]
	}`)

	requireCode(t, response, 200)
	requireJSONContent(t, response)
	requireCalls(t, commands.calls, "ack")
	if commands.ack.Token != channel.Token("x-token") {
		t.Fatalf("token = %q, want x-token", commands.ack.Token)
	}
	requireNeedOrder(t, commands.ack.DeclaredNeeds, "x12", "x13")
	if commands.ack.DeclaredNeeds[0].Target != registry.NodeKey("x17") {
		t.Fatalf("target = %q, want x17", commands.ack.DeclaredNeeds[0].Target)
	}
	requireBody(t, response, map[string]any{"resolved": true, "routed": false})
}

func TestAckRejectsMissingToken(t *testing.T) {
	commands := &recordingCommands{}

	response := serve(t, commands, "POST", "/ack", `{
		"lease_id": "x16",
		"declared_needs": []
	}`)

	requireCode(t, response, 400)
	requireCalls(t, commands.calls)
	requireBody(t, response, map[string]any{"error": "invalid_input"})
}

func TestAckRejectsNackOnlyFailurePayload(t *testing.T) {
	commands := &recordingCommands{}

	response := serve(t, commands, "POST", "/ack", `{
		"lease_id": "x16",
		"lease_token": "x-token",
		"failure_payload": {"reason": "x46"},
		"declared_needs": []
	}`)

	requireCode(t, response, 400)
	requireCalls(t, commands.calls)
	requireBody(t, response, map[string]any{"error": "invalid_input"})
}
