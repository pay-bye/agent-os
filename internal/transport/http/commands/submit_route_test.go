package commands

import (
	"testing"

	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestSubmitDispatchesCommandAndWritesRoutedResponse(t *testing.T) {
	commands := &recordingCommands{
		submitResult: kernel.SubmitResult{
			WorkItem: workitem.ID("x08"),
			Routed:   true,
			Channel:  registry.ChannelKey("x15"),
		},
	}

	response := serve(t, commands, "POST", "/submit", `{
		"work_item_id": "x08",
		"item_kind": "x08",
		"payload": {"value": "x75"},
		"declared_needs": [
			{"need_kind": "x12", "target_node": "x17", "payload": {"target_node": "x98", "value": "first"}},
			{"need_kind": "x13", "payload": {"value": "second"}}
		]
	}`)

	requireCode(t, response, 200)
	requireJSONContent(t, response)
	requireCalls(t, commands.calls, "submit")
	requireSubmitInput(t, commands.submit)
	requireBody(t, response, map[string]any{
		"work_item_id": "x08",
		"routed":       true,
		"channel_key":  "x15",
	})
}

func TestSubmitRejectsUnknownRequestFields(t *testing.T) {
	commands := &recordingCommands{}

	response := serve(t, commands, "POST", "/submit", `{
		"work_item_id": "x08",
		"item_kind": "x08",
		"payload": {},
		"declared_needs": [],
		"extra": true
	}`)

	requireCode(t, response, 400)
	requireCalls(t, commands.calls)
	requireBody(t, response, map[string]any{"error": "invalid_input"})
}

func TestSubmitRejectsMalformedPayloadObjects(t *testing.T) {
	commands := &recordingCommands{}

	response := serve(t, commands, "POST", "/submit", `{
		"work_item_id": "x08",
		"item_kind": "x08",
		"payload": [],
		"declared_needs": []
	}`)

	requireCode(t, response, 400)
	requireCalls(t, commands.calls)
	requireBody(t, response, map[string]any{"error": "invalid_input"})
}

func requireSubmitInput(t *testing.T, input kernel.SubmitInput) {
	t.Helper()

	if input.ID != workitem.ID("x08") {
		t.Fatalf("work item = %q", input.ID)
	}
	if input.Kind != registry.ItemKindKey("x08") {
		t.Fatalf("item kind = %q", input.Kind)
	}
	if string(input.Payload) != `{"value":"x75"}` {
		t.Fatalf("payload = %s", input.Payload)
	}
	requireNeedOrder(t, input.DeclaredNeeds, "x12", "x13")
	if input.DeclaredNeeds[0].Target != registry.NodeKey("x17") {
		t.Fatalf("target = %q, want x17", input.DeclaredNeeds[0].Target)
	}
	if input.DeclaredNeeds[1].Target != "" {
		t.Fatalf("unaddressed target = %q, want empty", input.DeclaredNeeds[1].Target)
	}
}
