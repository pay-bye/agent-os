package payloads

import "github.com/pay-bye/agent-os/internal/registry"

const InstructionAppliedKind registry.JournalEventKindKey = "x47"

func InstructionApplied(input InstructionOutcomeInput) ([]byte, error) {
	return marshalInstructionOutcome(input)
}
