package registry

import (
	"errors"
	"testing"
)

func TestNewRoutingRuleRejectsInvalidInput(t *testing.T) {
	for _, test := range []invalidRoutingRuleCase{
		{
			name: "empty need",
			input: validRoutingRuleInput(func(input *RoutingRuleInput) {
				input.NeedKind = ""
			}),
			want: ErrEmptyRoutingNeedKind,
		},
		{
			name: "empty node",
			input: validRoutingRuleInput(func(input *RoutingRuleInput) {
				input.Node = ""
			}),
			want: ErrEmptyRoutingNode,
		},
		{
			name: "missing order",
			input: validRoutingRuleInput(func(input *RoutingRuleInput) {
				input.Order = 0
			}),
			want: ErrInvalidRoutingOrder,
		},
	} {
		assertInvalidRoutingRule(t, test)
	}
}

func TestNewRoutingRulePreservesFields(t *testing.T) {
	rule, err := NewRoutingRule(validRoutingRuleInput())
	if err != nil {
		t.Fatal(err)
	}

	if rule.NeedKind() != NeedKindKey("x12") {
		t.Fatalf("need kind = %q, want x12", rule.NeedKind())
	}
	if rule.Node() != NodeKey("x17") {
		t.Fatalf("node = %q, want x17", rule.Node())
	}
	if rule.Order() != 1 {
		t.Fatalf("order = %d, want 1", rule.Order())
	}
}

func validRoutingRuleInput(changes ...func(*RoutingRuleInput)) RoutingRuleInput {
	input := RoutingRuleInput{
		NeedKind: NeedKindKey("x12"),
		Node:     NodeKey("x17"),
		Order:    1,
	}
	for _, change := range changes {
		change(&input)
	}
	return input
}

type invalidRoutingRuleCase struct {
	name  string
	input RoutingRuleInput
	want  error
}

func assertInvalidRoutingRule(t *testing.T, test invalidRoutingRuleCase) {
	t.Helper()

	t.Run(test.name, func(t *testing.T) {
		_, err := NewRoutingRule(test.input)

		if !errors.Is(err, test.want) {
			t.Fatalf("error = %v, want %v", err, test.want)
		}
	})
}
