package payloads

import "github.com/pay-bye/agent-os/internal/registry"

const ItemRoutedKind registry.JournalEventKindKey = "x42"

type RoutedItem struct {
	NeedKind  registry.NeedKindKey
	Node      registry.Node
	RuleOrder int
}

func ItemRouted(item RoutedItem) ([]byte, error) {
	payload := map[string]any{
		"need_kind":   item.NeedKind.String(),
		"node_key":    item.Node.Key().String(),
		"channel_key": item.Node.Channel().String(),
	}
	if item.RuleOrder > 0 {
		payload["routing_rule_order"] = item.RuleOrder
	}
	return marshal(payload)
}
