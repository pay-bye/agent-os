package registry

import (
	"context"
	"errors"
	"testing"

	records "github.com/pay-bye/agent-os/internal/registry"
)

func TestFindRoutingRulesReturnsNamedNoRoute(t *testing.T) {
	reader := &Store{
		queryRows: func(context.Context, string, ...any) (rowsScanner, error) {
			return &rowsValues{}, nil
		},
	}

	_, err := reader.FindRoutingRules(context.Background(), records.NeedKindKey("x14"))

	if !errors.Is(err, records.ErrNoRoute) {
		t.Fatalf("error = %v, want no route", err)
	}
}

func TestFindRoutingRulesMapsOrderedRows(t *testing.T) {
	reader := &Store{
		queryRows: func(context.Context, string, ...any) (rowsScanner, error) {
			return &rowsValues{rows: [][]any{
				{"x12", "x17", int64(1)},
				{"x12", "x18", int64(2)},
			}}, nil
		},
	}

	rules, err := reader.FindRoutingRules(context.Background(), records.NeedKindKey("x12"))
	if err != nil {
		t.Fatal(err)
	}

	if len(rules) != 2 {
		t.Fatalf("rule count = %d, want 2", len(rules))
	}
	if rules[1].Node() != records.NodeKey("x18") {
		t.Fatalf("second node = %q, want x18", rules[1].Node())
	}
}
