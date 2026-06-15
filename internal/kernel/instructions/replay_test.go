package instructions

import (
	"reflect"
	"testing"

	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/journal/payloads"
)

func TestReplayResultUsesJournalPayloads(t *testing.T) {
	record := ReplayRecord{
		ID:       "x70",
		EventIDs: []string{"x80", "x81"},
	}
	instruction := recordedEvent(t, "x80", payloads.InstructionAppliedKind, `{
		"instruction_id":"x70",
		"operation":"route_outstanding",
		"affected_ids":["x08"],
		"affected_count":1,
		"preconditions":["work_item_exists"],
		"result":"applied",
		"appended_at":"2026-05-18T12:00:00Z"
	}`, 1)
	routing := recordedEvent(t, "x81", payloads.ItemRoutedKind, `{"work_item_id":"x08"}`, 2)

	result, err := ReplayResult(record, []journal.Event{instruction, routing})
	record.EventIDs[0] = "mutated"
	if err != nil {
		t.Fatal(err)
	}

	if result.ID != ID("x70") {
		t.Fatalf("id = %q, want x70", result.ID)
	}
	if result.Result != Applied {
		t.Fatalf("result = %q, want applied", result.Result)
	}
	if !reflect.DeepEqual(result.EventIDs, []string{"x80", "x81"}) {
		t.Fatalf("events = %v, want x80/x81", result.EventIDs)
	}
	if !reflect.DeepEqual(result.AffectedIDs, []string{"x08"}) {
		t.Fatalf("affected = %v, want x08", result.AffectedIDs)
	}
}

func TestReplayResultRejectsMismatchedOutcomeID(t *testing.T) {
	record := ReplayRecord{ID: "x70", EventIDs: []string{"x80"}}
	event := recordedEvent(t, "x80", payloads.InstructionAppliedKind, `{
		"instruction_id":"x71",
		"result":"applied"
	}`, 1)

	_, err := ReplayResult(record, []journal.Event{event})

	if err == nil {
		t.Fatal("expected replay id mismatch")
	}
}

func TestReplayResultRejectsUnknownOutcome(t *testing.T) {
	record := ReplayRecord{ID: "x70", EventIDs: []string{"x80"}}
	event := recordedEvent(t, "x80", payloads.InstructionAppliedKind, `{
		"instruction_id":"x70",
		"result":"unknown"
	}`, 1)

	_, err := ReplayResult(record, []journal.Event{event})

	if err == nil {
		t.Fatal("expected unknown outcome result")
	}
}
