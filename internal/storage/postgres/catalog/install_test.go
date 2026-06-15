package catalog

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/internal/storage/postgres"
)

func TestInstallWritesRecordsInDependencyOrder(t *testing.T) {
	executor := &recordingExecutor{}
	snapshot := postgres.Catalog{
		Schemas: []postgres.SchemaRecord{{Key: "x01", Document: []byte(`{"type":"object"}`)}},
		Items:   []postgres.ItemRecord{{Key: "x08", Schema: "x01", Description: "x21"}},
		Needs:   []postgres.NeedRecord{{Key: "x12", Schema: "x01", HasSchema: true, Description: "x22"}},
		Nodes: []postgres.NodeRecord{{
			Key:          "x17",
			Description:  "x23",
			Accepts:      []string{"x12"},
			Channel:      "x17",
			ChannelLabel: "x23",
		}},
		Routes: []postgres.RouteRecord{{Need: "x12", Node: "x17", Order: 1}},
	}

	err := Install(context.Background(), executor, snapshot)

	if err != nil {
		t.Fatal(err)
	}
	requireStatementOrder(t, executor.statements, []string{
		"INSERT INTO schema_documents",
		"INSERT INTO item_kinds",
		"INSERT INTO need_kinds",
		"INSERT INTO nodes",
		"INSERT INTO channels",
		"INSERT INTO node_capabilities",
		"INSERT INTO routing_rules",
	})
}

func requireStatementOrder(t *testing.T, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("statements = %v, want %v", got, want)
	}
	for index, wantPart := range want {
		if !strings.Contains(got[index], wantPart) {
			t.Fatalf("statement[%d] = %q, want %q", index, got[index], wantPart)
		}
	}
}

type recordingExecutor struct {
	statements []string
	args       [][]any
}

func (e *recordingExecutor) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	copied := append([]any(nil), args...)
	e.statements = append(e.statements, query)
	e.args = append(e.args, copied)
	return nil, nil
}
