package payloads

import "github.com/pay-bye/agent-os/internal/registry"

const WorkItemDroppedKind registry.JournalEventKindKey = "x49"

func WorkItemDropped(input InstructionOutcomeInput) ([]byte, error) {
	return marshalInstructionOutcome(input)
}
