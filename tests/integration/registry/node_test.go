//go:build integration

package registry_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/pay-bye/agent-os/internal/registry"
	registrystore "github.com/pay-bye/agent-os/internal/storage/postgres/registry"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

func TestReaderFindsNodeRecords(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	insertRecords(t, ctx, tx)
	reader := registrystore.New(tx)

	node, err := reader.FindNode(ctx, registry.NodeKey("x17"))
	if err != nil {
		t.Fatal(err)
	}
	channel, err := reader.FindChannel(ctx, registry.ChannelKey("x15"))
	if err != nil {
		t.Fatal(err)
	}

	requireNode(t, node)
	requireChannel(t, channel)
}

func TestReaderReturnsNamedNodeNotFound(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	reader := registrystore.New(tx)

	_, err := reader.FindNode(ctx, registry.NodeKey("x60"))

	if !registry.IsNotFound(err) {
		t.Fatalf("expected registry not-found error, got %v", err)
	}
}

func requireNode(t *testing.T, node registry.Node) {
	t.Helper()

	if node.Key() != registry.NodeKey("x17") {
		t.Fatalf("node key = %q, want x17", node.Key())
	}
	if node.Description() != "First" {
		t.Fatalf("description = %q, want First", node.Description())
	}
	if node.Channel() != registry.ChannelKey("x15") {
		t.Fatalf("channel = %q, want x15", node.Channel())
	}
	capabilities := node.Capabilities()
	if len(capabilities) != 2 {
		t.Fatalf("capability count = %d, want 2", len(capabilities))
	}
	if capabilities[0] != registry.NeedKindKey("x12") || capabilities[1] != registry.NeedKindKey("x13") {
		t.Fatalf("capabilities = %v, want [x12 x13]", capabilities)
	}
}

func requireChannel(t *testing.T, channel registry.Channel) {
	t.Helper()

	if channel.Key() != registry.ChannelKey("x15") {
		t.Fatalf("channel key = %q, want x15", channel.Key())
	}
	if channel.Node() != registry.NodeKey("x17") {
		t.Fatalf("node key = %q, want x17", channel.Node())
	}
	if channel.Description() != "First" {
		t.Fatalf("description = %q, want First", channel.Description())
	}
}

func insertRecords(t *testing.T, ctx context.Context, tx *sql.Tx) {
	t.Helper()

	_, err := tx.ExecContext(ctx, `
INSERT INTO schema_documents (key, document) VALUES ('x01', '{"title":"First"}');
INSERT INTO need_kinds (key, schema_key, description) VALUES ('x12', 'x01', 'First');
INSERT INTO need_kinds (key, schema_key, description) VALUES ('x13', 'x01', 'Second');
INSERT INTO nodes (key, description) VALUES ('x17', 'First');
INSERT INTO channels (key, node_key, description) VALUES ('x15', 'x17', 'First');
INSERT INTO node_capabilities (node_key, need_kind_key) VALUES ('x17', 'x12');
INSERT INTO node_capabilities (node_key, need_kind_key) VALUES ('x17', 'x13');
`)
	if err != nil {
		t.Fatal(err)
	}
}
