package registry

import (
	"context"
	"database/sql"
	records "github.com/pay-bye/agent-os/internal/registry"
)

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type Store struct {
	query     queryFunc
	queryRows rowsQueryFunc
}

func (r *Store) FindChannel(ctx context.Context, key records.ChannelKey) (records.Channel, error) {
	row := r.query(ctx, `SELECT node_key, description FROM channels WHERE key = $1`, key.String())
	return channelFromRow(key, row)
}

func (r *Store) FindItemKind(ctx context.Context, key records.ItemKindKey) (records.ItemKind, error) {
	row := r.query(ctx, `SELECT schema_key, description FROM item_kinds WHERE key = $1`, key.String())
	return itemKindFromRow(key, row)
}

func (r *Store) FindJournalEventKind(
	ctx context.Context,
	key records.JournalEventKindKey,
) (records.JournalEventKind, error) {
	row := r.query(ctx, `SELECT schema_key, description FROM journal_event_kinds WHERE key = $1`, key.String())
	return journalEventKindFromRow(key, row)
}

func (r *Store) FindNeedKind(ctx context.Context, key records.NeedKindKey) (records.NeedKind, error) {
	row := r.query(ctx, `SELECT schema_key, description FROM need_kinds WHERE key = $1`, key.String())
	return needKindFromRow(key, row)
}

func (r *Store) FindNode(ctx context.Context, key records.NodeKey) (records.Node, error) {
	row := r.query(ctx, `
SELECT n.description, c.key
FROM nodes n
JOIN channels c ON c.node_key = n.key
WHERE n.key = $1`, key.String())
	description, channel, err := nodeRowValues(key, row)
	if err != nil {
		return records.Node{}, err
	}
	capabilities, err := r.nodeCapabilities(ctx, key)
	if err != nil {
		return records.Node{}, err
	}
	return records.NewNode(records.NodeInput{
		Key:          key,
		Description:  description,
		Channel:      channel,
		Capabilities: capabilities,
	})
}

func (r *Store) nodeCapabilities(ctx context.Context, key records.NodeKey) ([]records.NeedKindKey, error) {
	rows, err := r.queryRows(ctx, `
SELECT need_kind_key
FROM node_capabilities
WHERE node_key = $1
ORDER BY need_kind_key`, key.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var capabilities []records.NeedKindKey
	for rows.Next() {
		var capability string
		if err := rows.Scan(&capability); err != nil {
			return nil, err
		}
		capabilities = append(capabilities, records.NeedKindKey(capability))
	}
	return capabilities, rows.Err()
}

func (r *Store) FindRoutingRules(ctx context.Context, key records.NeedKindKey) ([]records.RoutingRule, error) {
	rows, err := r.queryRows(ctx, `
SELECT need_kind_key, node_key, rule_order
FROM routing_rules
WHERE need_kind_key = $1
ORDER BY rule_order`, key.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules, err := scanRoutingRules(rows)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return nil, records.ErrNoRoute
	}
	return rules, nil
}

func (r *Store) FindSchemaDocument(ctx context.Context, key records.SchemaKey) (records.SchemaDocument, error) {
	row := r.query(ctx, `SELECT document FROM schema_documents WHERE key = $1`, key.String())
	return schemaDocumentFromRow(key, row)
}

func New(db queryer) *Store {
	return &Store{
		query: func(ctx context.Context, query string, args ...any) rowScanner {
			return db.QueryRowContext(ctx, query, args...)
		},
		queryRows: func(ctx context.Context, query string, args ...any) (rowsScanner, error) {
			return db.QueryContext(ctx, query, args...)
		},
	}
}
