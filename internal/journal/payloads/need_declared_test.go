package payloads

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestNeedDeclaredIncludesTargetNode(t *testing.T) {
	need := declaredNeed(t)

	body, err := NeedDeclared(need)
	got := decode(t, body, err)

	if got["need_kind"] != "x12" || got["target_node"] != "x17" {
		t.Fatalf("need declared payload = %+v", got)
	}
	payload := got["payload"].(map[string]any)
	if payload["case"] != "x48" {
		t.Fatalf("need payload = %+v", payload)
	}
}

func TestNeedFromEventPreservesKindAndTarget(t *testing.T) {
	event := recordedEvent(t, `{"need_kind":"x12","target_node":"x17","payload":{"case":"x48"}}`)

	need, err := NeedFromEvent(event)

	if err != nil {
		t.Fatal(err)
	}
	if need.Kind != registry.NeedKindKey("x12") || need.Target != registry.NodeKey("x17") {
		t.Fatalf("need = %+v", need)
	}
}

func TestNeedFromEventRejectsMalformedPayload(t *testing.T) {
	event := recordedEvent(t, `{"payload":{"case":"x48"}}`)

	_, err := NeedFromEvent(event)

	if !errors.Is(err, ErrMalformedNeed) {
		t.Fatalf("error = %v, want malformed need", err)
	}
}

func declaredNeed(t *testing.T) workitem.DeclaredNeed {
	t.Helper()

	need, err := workitem.NewDeclaredNeed(workitem.DeclaredNeedInput{
		Kind:    "x12",
		Target:  "x17",
		Payload: []byte(`{"case":"x48"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return need
}

func recordedEvent(t *testing.T, payload string) journal.Event {
	t.Helper()

	event, err := journal.NewRecordedEvent(journal.EventInput{
		ID:         "x22",
		Coordinate: journal.WorkItemCoordinate("x08"),
		Kind:       NeedDeclaredKind,
		AppendedAt: time.Date(2026, 5, 18, 12, 1, 0, 0, time.UTC),
		Payload:    []byte(payload),
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	return event
}

func decode(t *testing.T, body []byte, err error) map[string]any {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}
