package registry

import (
	"context"
	"testing"

	records "github.com/pay-bye/agent-os/internal/registry"
)

func TestFindNodeReturnsNamedNotFound(t *testing.T) {
	reader := &Store{query: func(context.Context, string, ...any) rowScanner {
		return missingRow{}
	}}

	_, err := reader.FindNode(context.Background(), records.NodeKey("x60"))

	if !records.IsNotFound(err) {
		t.Fatalf("expected registry not-found error, got %v", err)
	}
}

func TestFindNodeMapsCapabilities(t *testing.T) {
	reader := &Store{
		query: func(context.Context, string, ...any) rowScanner {
			return rowValues{values: []any{"First", "x15"}}
		},
		queryRows: func(context.Context, string, ...any) (rowsScanner, error) {
			return &rowsValues{rows: [][]any{{"x12"}, {"x13"}}}, nil
		},
	}

	node, err := reader.FindNode(context.Background(), records.NodeKey("x17"))
	if err != nil {
		t.Fatal(err)
	}

	if len(node.Capabilities()) != 2 {
		t.Fatalf("capability count = %d, want 2", len(node.Capabilities()))
	}
}
