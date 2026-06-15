package payloads

import (
	"encoding/json"
	"errors"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

const NeedDeclaredKind registry.JournalEventKindKey = "x41"

var ErrMalformedNeed = errors.New("need declaration event is malformed")

type Need struct {
	Kind   registry.NeedKindKey
	Target registry.NodeKey
}

type needDocument struct {
	Kind   string `json:"need_kind"`
	Target string `json:"target_node"`
}

func NeedDeclared(need workitem.DeclaredNeed) ([]byte, error) {
	payload := map[string]any{
		"need_kind": need.Kind().String(),
		"payload":   raw(need.Payload()),
	}
	if need.Target().String() != "" {
		payload["target_node"] = need.Target().String()
	}
	return marshal(payload)
}

func NeedFromEvent(event journal.Event) (Need, error) {
	var document needDocument
	if err := json.Unmarshal(event.Payload(), &document); err != nil {
		return Need{}, ErrMalformedNeed
	}
	if document.Kind == "" {
		return Need{}, ErrMalformedNeed
	}
	return Need{
		Kind:   registry.NeedKindKey(document.Kind),
		Target: registry.NodeKey(document.Target),
	}, nil
}
