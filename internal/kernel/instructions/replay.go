package instructions

import (
	"errors"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/journal/payloads"
)

type ReplayRecord struct {
	ID       ID
	EventIDs []string
}

func ReplayResult(record ReplayRecord, events []journal.Event) (Result, error) {
	result := Result{
		ID:       record.ID,
		EventIDs: copyStrings(record.EventIDs),
	}
	for _, event := range events {
		payload, ok, err := payloadFromEvent(record.ID.String(), event)
		if err != nil {
			return Result{}, err
		}
		if !ok {
			continue
		}
		if err := applyReplayPayload(&result, payload); err != nil {
			return Result{}, err
		}
	}
	if result.Result == "" {
		return Result{}, errors.New("instruction journal outcome is missing")
	}
	return result, nil
}

func payloadFromEvent(id string, event journal.Event) (payloads.InstructionOutcomeDocument, bool, error) {
	return payloads.InstructionOutcomeFromEvent(id, event)
}

func applyReplayPayload(result *Result, payload payloads.InstructionOutcomeDocument) error {
	value := ResultValue(payload.Result)
	if value != Applied && value != PreconditionFailed {
		return errors.New("instruction journal outcome result is unknown")
	}
	if result.Result != "" && result.Result != value {
		return errors.New("instruction journal outcome result mismatch")
	}
	result.Result = value
	result.AffectedIDs = append(result.AffectedIDs, payload.AffectedIDs...)
	if payload.FailedPrecondition != "" {
		result.FailedPrecondition = payload.FailedPrecondition
	}
	return nil
}
