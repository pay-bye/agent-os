package catalog

import (
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/journal/payloads"
	"github.com/pay-bye/agent-os/internal/registry"
)

func TestExclusionClearEventIdentifiesNode(t *testing.T) {
	appendedAt := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)

	event, err := exclusionClearEvent("x17", "x80", appendedAt)
	if err != nil {
		t.Fatal(err)
	}

	if event.ID != journal.EventID("x80") {
		t.Fatalf("event identity = %q, want x80", event.ID)
	}
	if event.Coordinate != journal.NodeCoordinate(registry.NodeKey("x17")) {
		t.Fatalf("coordinate = %+v, want node x17", event.Coordinate)
	}
	if event.Kind != payloads.ExclusionClearKind {
		t.Fatalf("kind = %s, want x46", event.Kind)
	}
	if string(event.Payload) != `{"node_key":"x17"}` {
		t.Fatalf("payload = %s, want node key", event.Payload)
	}
	if !event.AppendedAt.Equal(appendedAt) {
		t.Fatalf("appended at = %s, want %s", event.AppendedAt, appendedAt)
	}
}
