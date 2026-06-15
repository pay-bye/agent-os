package payloads

import (
	"errors"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestItemSubmittedIdentifiesWorkItemAndKind(t *testing.T) {
	submission := submittedItem(t)

	body, err := ItemSubmitted(submission)
	got := decode(t, body, err)

	if got["work_item_id"] != "x08" || got["item_kind"] != "x10" {
		t.Fatalf("item submitted payload = %+v", got)
	}
}

func TestNeedAckedIdentifiesLease(t *testing.T) {
	body, err := NeedAcked(Resolution{Lease: leasedWorkItem(t)})
	got := decode(t, body, err)

	if got["lease_id"] != "x16" || got["work_item_id"] != "x08" {
		t.Fatalf("resolution payload = %+v", got)
	}
}

func TestNeedNackedIncludesFailurePayload(t *testing.T) {
	body, err := NeedNacked(Resolution{
		Lease:          leasedWorkItem(t),
		FailurePayload: []byte(`{"reason":"x91"}`),
	})
	got := decode(t, body, err)

	if got["lease_id"] != "x16" || got["work_item_id"] != "x08" {
		t.Fatalf("resolution payload = %+v", got)
	}
	failure := got["failure_payload"].(map[string]any)
	if failure["reason"] != "x91" {
		t.Fatalf("failure payload = %+v", failure)
	}
}

func TestExclusionSetIdentifiesNode(t *testing.T) {
	body, err := ExclusionSet("x17")
	got := decode(t, body, err)

	if got["node_key"] != "x17" {
		t.Fatalf("exclusion payload = %+v", got)
	}
}

func TestExclusionClearIdentifiesNode(t *testing.T) {
	body, err := ExclusionClear("x17")
	got := decode(t, body, err)

	if got["node_key"] != "x17" {
		t.Fatalf("exclusion payload = %+v", got)
	}
}

func TestInstructionOutcomeFromEventSkipsUnrelatedDocuments(t *testing.T) {
	event := instructionEvent(t, []byte(`{"event":"x48"}`))

	_, ok, err := InstructionOutcomeFromEvent("x70", event)

	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected unrelated document to be skipped")
	}
}

func TestInstructionOutcomeFromEventRejectsIDMismatch(t *testing.T) {
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

	_, _, err = InstructionOutcomeFromEvent("x71", event)

	if !errors.Is(err, ErrInstructionOutcomeIDMismatch) {
		t.Fatalf("error = %v, want instruction outcome id mismatch", err)
	}
}

func submittedItem(t *testing.T) workitem.Submission {
	t.Helper()

	submission, err := workitem.NewSubmission(workitem.SubmissionInput{
		ID:      "x08",
		Kind:    "x10",
		Payload: []byte(`{"case":"x48"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return submission
}

func leasedWorkItem(t *testing.T) channel.Lease {
	t.Helper()

	lease, err := channel.NewLease(channel.LeaseInput{
		ID:        "x16",
		Entry:     "x31",
		Channel:   registry.ChannelKey("x15"),
		WorkItem:  "x08",
		GrantedAt: time.Date(2026, 5, 18, 12, 1, 0, 0, time.UTC),
		ExpiresAt: time.Date(2026, 5, 18, 12, 6, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	return lease
}
