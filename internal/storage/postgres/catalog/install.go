package catalog

import (
	"context"
	"database/sql"

	"github.com/pay-bye/agent-os/internal/storage/postgres"
)

type catalogExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func Install(ctx context.Context, executor catalogExecutor, catalog postgres.Catalog) error {
	if err := insertSchemas(ctx, executor, catalog.Schemas); err != nil {
		return err
	}
	if err := insertItems(ctx, executor, catalog.Items); err != nil {
		return err
	}
	if err := insertNeeds(ctx, executor, catalog.Needs); err != nil {
		return err
	}
	if err := insertNodes(ctx, executor, catalog.Nodes); err != nil {
		return err
	}
	return insertRoutes(ctx, executor, catalog.Routes)
}

func insertSchemas(ctx context.Context, executor catalogExecutor, records []postgres.SchemaRecord) error {
	for _, record := range records {
		_, err := executor.ExecContext(ctx, `
INSERT INTO schema_documents (key, document)
VALUES ($1, $2)
ON CONFLICT (key) DO NOTHING`, record.Key, string(record.Document))
		if err != nil {
			return err
		}
	}
	return nil
}

func insertItems(ctx context.Context, executor catalogExecutor, records []postgres.ItemRecord) error {
	for _, record := range records {
		_, err := executor.ExecContext(ctx, `
INSERT INTO item_kinds (key, schema_key, description)
VALUES ($1, $2, $3)
ON CONFLICT (key) DO NOTHING`, record.Key, record.Schema, record.Description)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertNeeds(ctx context.Context, executor catalogExecutor, records []postgres.NeedRecord) error {
	for _, record := range records {
		var schema any
		if record.HasSchema {
			schema = record.Schema
		}
		_, err := executor.ExecContext(ctx, `
INSERT INTO need_kinds (key, schema_key, description)
VALUES ($1, $2, $3)
ON CONFLICT (key) DO NOTHING`, record.Key, schema, record.Description)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertNodes(ctx context.Context, executor catalogExecutor, records []postgres.NodeRecord) error {
	for _, record := range records {
		if err := insertNode(ctx, executor, record); err != nil {
			return err
		}
	}
	return nil
}

func insertNode(ctx context.Context, executor catalogExecutor, record postgres.NodeRecord) error {
	if _, err := executor.ExecContext(ctx, `
INSERT INTO nodes (key, description)
VALUES ($1, $2)
ON CONFLICT (key) DO NOTHING`, record.Key, record.Description); err != nil {
		return err
	}
	if _, err := executor.ExecContext(ctx, `
INSERT INTO channels (key, node_key, description)
VALUES ($1, $2, $3)
ON CONFLICT (key) DO NOTHING`, record.Channel, record.Key, record.ChannelLabel); err != nil {
		return err
	}
	for _, need := range record.Accepts {
		if _, err := executor.ExecContext(ctx, `
INSERT INTO node_capabilities (node_key, need_kind_key)
VALUES ($1, $2)
ON CONFLICT (node_key, need_kind_key) DO NOTHING`, record.Key, need); err != nil {
			return err
		}
	}
	return nil
}

func insertRoutes(ctx context.Context, executor catalogExecutor, records []postgres.RouteRecord) error {
	for _, record := range records {
		_, err := executor.ExecContext(ctx, `
INSERT INTO routing_rules (need_kind_key, node_key, rule_order)
VALUES ($1, $2, $3)
ON CONFLICT (need_kind_key, rule_order) DO NOTHING`, record.Need, record.Node, record.Order)
		if err != nil {
			return err
		}
	}
	return nil
}
