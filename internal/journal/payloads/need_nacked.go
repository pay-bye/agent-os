package payloads

import "github.com/pay-bye/agent-os/internal/registry"

const NeedNackedKind registry.JournalEventKindKey = "x44"

func NeedNacked(resolution Resolution) ([]byte, error) {
	return marshalResolution(resolution)
}
