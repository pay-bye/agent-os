package payloads

import "github.com/pay-bye/agent-os/internal/registry"

const ExclusionSetKind registry.JournalEventKindKey = "x45"

func ExclusionSet(node registry.NodeKey) ([]byte, error) {
	return marshalExclusion(node)
}
