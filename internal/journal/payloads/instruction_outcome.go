package payloads

import (
	"encoding/json"
	"errors"
	"github.com/pay-bye/agent-os/internal/journal"
	"time"
)

var ErrInstructionOutcomeIDMismatch = errors.New("instruction journal outcome id mismatch")

type InstructionOutcomeInput struct {
	ID                 string
	Operation          string
	AuditFields        map[string]any
	AffectedIDs        []string
	FailedPrecondition string
	Result             string
	Preconditions      []string
	AppendedAt         time.Time
}

type InstructionOutcomeDocument struct {
	ID                 string   `json:"instruction_id"`
	Result             string   `json:"result"`
	AffectedIDs        []string `json:"affected_ids"`
	FailedPrecondition string   `json:"failed_precondition"`
}

func InstructionOutcomeFromEvent(id string, event journal.Event) (InstructionOutcomeDocument, bool, error) {
	var outcome InstructionOutcomeDocument
	if err := json.Unmarshal(event.Payload(), &outcome); err != nil {
		return InstructionOutcomeDocument{}, false, err
	}
	if outcome.ID == "" {
		return InstructionOutcomeDocument{}, false, nil
	}
	if outcome.ID != id {
		return InstructionOutcomeDocument{}, false, ErrInstructionOutcomeIDMismatch
	}
	return outcome, true, nil
}

func marshalInstructionOutcome(input InstructionOutcomeInput) ([]byte, error) {
	affected := copyStrings(input.AffectedIDs)
	preconditions := copyStrings(input.Preconditions)
	value := copyFields(input.AuditFields)
	value["instruction_id"] = input.ID
	value["operation"] = input.Operation
	value["affected_ids"] = affected
	value["affected_count"] = len(affected)
	value["preconditions"] = preconditions
	value["result"] = input.Result
	value["appended_at"] = input.AppendedAt.Format(time.RFC3339)
	if input.FailedPrecondition != "" {
		value["failed_precondition"] = input.FailedPrecondition
	}
	return marshal(value)
}

func copyFields(fields map[string]any) map[string]any {
	value := make(map[string]any, len(fields)+7)
	for key, field := range fields {
		value[key] = copyField(field)
	}
	return value
}

func copyField(field any) any {
	values, ok := field.([]string)
	if !ok {
		return field
	}
	return copyStrings(values)
}

func copyStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string(nil), values...)
}
