package payloads

import "github.com/pay-bye/agent-os/internal/registry"

const ExclusionClearKind registry.JournalEventKindKey = "x46"

func ExclusionClear(node registry.NodeKey) ([]byte, error) {
	return marshalExclusion(node)
}
