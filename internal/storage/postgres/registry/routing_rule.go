package registry

import (
	records "github.com/pay-bye/agent-os/internal/registry"
)

type routingRuleInput struct {
	needKind string
	node     string
	order    int64
}

func scanRoutingRules(rows rowsScanner) ([]records.RoutingRule, error) {
	var rules []records.RoutingRule
	for rows.Next() {
		rule, err := scanRoutingRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func scanRoutingRule(row rowScanner) (records.RoutingRule, error) {
	var input routingRuleInput
	if err := row.Scan(&input.needKind, &input.node, &input.order); err != nil {
		return records.RoutingRule{}, err
	}
	return records.NewRoutingRule(records.RoutingRuleInput{
		NeedKind: records.NeedKindKey(input.needKind),
		Node:     records.NodeKey(input.node),
		Order:    int(input.order),
	})
}
