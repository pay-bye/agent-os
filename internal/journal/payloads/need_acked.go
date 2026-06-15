package payloads

import "github.com/pay-bye/agent-os/internal/registry"

const NeedAckedKind registry.JournalEventKindKey = "x43"

func NeedAcked(resolution Resolution) ([]byte, error) {
	return marshalResolution(resolution)
}
