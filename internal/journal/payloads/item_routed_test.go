package payloads

import (
	"testing"

	"github.com/pay-bye/agent-os/internal/registry"
)

func TestItemRoutedHandlesDefaultAndAddressedRoutes(t *testing.T) {
	defaultBody, defaultErr := ItemRouted(RoutedItem{
		NeedKind:  "x12",
		Node:      routeNode(t),
		RuleOrder: 2,
	})
	defaultRoute := decode(t, defaultBody, defaultErr)
	addressedBody, addressedErr := ItemRouted(RoutedItem{
		NeedKind: "x12",
		Node:     routeNode(t),
	})
	addressedRoute := decode(t, addressedBody, addressedErr)

	if defaultRoute["routing_rule_order"] != float64(2) {
		t.Fatalf("default route payload = %+v", defaultRoute)
	}
	if _, ok := addressedRoute["routing_rule_order"]; ok {
		t.Fatalf("addressed route payload = %+v", addressedRoute)
	}
	if addressedRoute["node_key"] != "x17" || addressedRoute["channel_key"] != "x17" {
		t.Fatalf("addressed route payload = %+v", addressedRoute)
	}
}

func routeNode(t *testing.T) registry.Node {
	t.Helper()

	node, err := registry.NewNode(registry.NodeInput{
		Key:          "x17",
		Description:  "x23",
		Channel:      "x17",
		Capabilities: []registry.NeedKindKey{"x12"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return node
}
