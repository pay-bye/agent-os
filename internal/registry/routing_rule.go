package registry

import (
	"errors"
)

var (
	ErrEmptyRoutingNeedKind = errors.New("routing need kind is empty")
	ErrEmptyRoutingNode     = errors.New("routing node is empty")
	ErrInvalidRoutingOrder  = errors.New("routing order must be positive")
	ErrNoRoute              = errors.New("routing rule not found")
)

type RoutingRuleInput struct {
	NeedKind NeedKindKey
	Node     NodeKey
	Order    int
}

type RoutingRule struct {
	needKind NeedKindKey
	node     NodeKey
	order    int
}

func NewRoutingRule(input RoutingRuleInput) (RoutingRule, error) {
	if err := validateRoutingRuleInput(input); err != nil {
		return RoutingRule{}, err
	}
	return RoutingRule{
		needKind: input.NeedKind,
		node:     input.Node,
		order:    input.Order,
	}, nil
}

func (r RoutingRule) NeedKind() NeedKindKey {
	return r.needKind
}

func (r RoutingRule) Node() NodeKey {
	return r.node
}

func (r RoutingRule) Order() int {
	return r.order
}

func validateRoutingRuleInput(input RoutingRuleInput) error {
	if blank(input.NeedKind.String()) {
		return ErrEmptyRoutingNeedKind
	}
	if blank(input.Node.String()) {
		return ErrEmptyRoutingNode
	}
	if input.Order <= 0 {
		return ErrInvalidRoutingOrder
	}
	return nil
}
