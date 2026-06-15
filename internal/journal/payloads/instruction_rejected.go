package payloads

import "github.com/pay-bye/agent-os/internal/registry"

const InstructionRejectedKind registry.JournalEventKindKey = "x48"

func InstructionRejected(input InstructionOutcomeInput) ([]byte, error) {
	return marshalInstructionOutcome(input)
}
