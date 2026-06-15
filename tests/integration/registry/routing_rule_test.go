//go:build integration

package registry_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/pay-bye/agent-os/internal/registry"
	registrystore "github.com/pay-bye/agent-os/internal/storage/postgres/registry"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

func TestRoutingRulesReturnOrderedNodeTargets(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	insertRoutingVocabulary(t, ctx, tx)
	reader := registrystore.New(tx)

	rules, err := reader.FindRoutingRules(ctx, registry.NeedKindKey("x12"))
	if err != nil {
		t.Fatal(err)
	}

	if len(rules) != 2 {
		t.Fatalf("rule count = %d, want 2", len(rules))
	}
	if rules[0].Node() != registry.NodeKey("x17") {
		t.Fatalf("first node = %q, want x17", rules[0].Node())
	}
	if rules[1].Node() != registry.NodeKey("x18") {
		t.Fatalf("second node = %q, want x18", rules[1].Node())
	}
}

func TestRoutingRulesRequireNodeCapability(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	insertRoutingVocabulary(t, ctx, tx)

	_, err := tx.ExecContext(ctx, `
INSERT INTO nodes (key, description) VALUES ('x19', 'Third');
INSERT INTO routing_rules (need_kind_key, node_key, rule_order)
VALUES ('x12', 'x19', 3);
`)

	if err == nil {
		t.Fatal("expected routing rule without capability to fail")
	}
}

func TestRoutingRulesReturnNamedNoRoute(t *testing.T) {
	ctx := context.Background()
	tx := postgresfixture.MigratedTransaction(t, ctx)
	insertRoutingVocabulary(t, ctx, tx)
	reader := registrystore.New(tx)

	_, err := reader.FindRoutingRules(ctx, registry.NeedKindKey("x14"))

	if !errors.Is(err, registry.ErrNoRoute) {
		t.Fatalf("error = %v, want no route", err)
	}
}

func insertRoutingVocabulary(t *testing.T, ctx context.Context, tx *sql.Tx) {
	t.Helper()

	_, err := tx.ExecContext(ctx, `
INSERT INTO schema_documents (key, document) VALUES ('x01', '{"title":"First"}');
INSERT INTO need_kinds (key, schema_key, description) VALUES ('x12', 'x01', 'First');
INSERT INTO need_kinds (key, schema_key, description) VALUES ('x14', 'x01', 'Absent');
INSERT INTO nodes (key, description) VALUES ('x17', 'First');
INSERT INTO nodes (key, description) VALUES ('x18', 'Second');
INSERT INTO channels (key, node_key, description) VALUES ('x15', 'x17', 'First');
INSERT INTO channels (key, node_key, description) VALUES ('x68', 'x18', 'Second');
INSERT INTO node_capabilities (node_key, need_kind_key) VALUES ('x17', 'x12');
INSERT INTO node_capabilities (node_key, need_kind_key) VALUES ('x18', 'x12');
INSERT INTO routing_rules (need_kind_key, node_key, rule_order) VALUES ('x12', 'x17', 1);
INSERT INTO routing_rules (need_kind_key, node_key, rule_order) VALUES ('x12', 'x18', 2);
`)
	if err != nil {
		t.Fatal(err)
	}
}
