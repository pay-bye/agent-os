package payloads

import "github.com/pay-bye/agent-os/internal/registry"

func marshalExclusion(node registry.NodeKey) ([]byte, error) {
	return marshal(map[string]any{"node_key": node.String()})
}
