package payloads

import (
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/journal"
)

func TestInstructionAppliedCopiesAuditFields(t *testing.T) {
	fields := map[string]any{"entries": []string{"x31", "x32"}}

	body, err := InstructionApplied(InstructionOutcomeInput{
		ID:            "x70",
		Operation:     "move_entries",
		AuditFields:   fields,
		AffectedIDs:   []string{"x31", "x32"},
		Result:        "applied",
		Preconditions: []string{"entry_in_source"},
		AppendedAt:    time.Date(2026, 5, 18, 12, 1, 0, 0, time.UTC),
	})
	got := decode(t, body, err)
	fields["entries"].([]string)[0] = "changed"

	if got["operation"] != "move_entries" || got["affected_count"] != float64(2) {
		t.Fatalf("instruction payload = %+v", got)
	}
	entries := got["entries"].([]any)
	if entries[0] != "x31" {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestInstructionRejectedIncludesFailedPrecondition(t *testing.T) {
	body, err := InstructionRejected(InstructionOutcomeInput{
		ID:                 "x70",
		Operation:          "release_expired_lease",
		FailedPrecondition: "lease_expired",
		Result:             "precondition_failed",
		AppendedAt:         time.Date(2026, 5, 18, 12, 1, 0, 0, time.UTC),
	})
	got := decode(t, body, err)

	if got["failed_precondition"] != "lease_expired" {
		t.Fatalf("instruction payload = %+v", got)
	}
}

func TestWorkItemDroppedIdentifiesAffectedWorkItem(t *testing.T) {
	body, err := WorkItemDropped(InstructionOutcomeInput{
		ID:          "x70",
		Operation:   "drop",
		AffectedIDs: []string{"x08"},
		Result:      "applied",
		AppendedAt:  time.Date(2026, 5, 18, 12, 1, 0, 0, time.UTC),
	})
	got := decode(t, body, err)

	if got["affected_count"] != float64(1) {
		t.Fatalf("instruction payload = %+v", got)
	}
}

func TestInstructionOutcomeFromEventRequiresMatchingID(t *testing.T) {
	body, err := InstructionApplied(InstructionOutcomeInput{
		ID:         "x70",
		Operation:  "drop",
		Result:     "applied",
		AppendedAt: time.Date(2026, 5, 18, 12, 1, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	event := instructionEvent(t, body)

	outcome, ok, err := InstructionOutcomeFromEvent("x70", event)

	if err != nil {
		t.Fatal(err)
	}
	if !ok || outcome.Result != "applied" {
		t.Fatalf("outcome = %+v, ok = %v", outcome, ok)
	}
}

func instructionEvent(t *testing.T, body []byte) journal.Event {
	t.Helper()

	event, err := journal.NewRecordedEvent(journal.EventInput{
		ID:         "x80",
		Coordinate: journal.WorkItemCoordinate("x08"),
		Kind:       InstructionAppliedKind,
		AppendedAt: time.Date(2026, 5, 18, 12, 1, 0, 0, time.UTC),
		Payload:    body,
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	return event
}
