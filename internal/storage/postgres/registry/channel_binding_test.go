package registry

import (
	"context"
	"testing"

	records "github.com/pay-bye/agent-os/internal/registry"
)

func TestFindChannelReturnsNamedNotFound(t *testing.T) {
	reader := &Store{query: func(context.Context, string, ...any) rowScanner {
		return missingRow{}
	}}

	_, err := reader.FindChannel(context.Background(), records.ChannelKey("x64"))

	if !records.IsNotFound(err) {
		t.Fatalf("expected registry not-found error, got %v", err)
	}
}
