package catalog

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/pay-bye/agent-os/internal/declaration"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/journal/payloads"
	"github.com/pay-bye/agent-os/internal/process/ids"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/storage/postgres"
	eventstore "github.com/pay-bye/agent-os/internal/storage/postgres/journal"
)

type Store struct {
	tx  *sql.Tx
	ids ids.Random
}

func (s *Store) Snapshot(ctx context.Context) (postgres.Catalog, error) {
	return Read(ctx, s.tx)
}

func (s *Store) Install(ctx context.Context, snapshot postgres.Catalog) error {
	return Install(ctx, s.tx, snapshot)
}

func (s *Store) Reconcile(ctx context.Context, delta declaration.Delta) error {
	if err := s.clearExclusions(ctx, delta.Clearances); err != nil {
		return err
	}
	for _, removal := range delta.Removals {
		if err := s.remove(ctx, removal); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) clearExclusions(ctx context.Context, refs []declaration.RecordRef) error {
	for _, ref := range refs {
		removed, err := s.deleteExclusion(ctx, ref.Key)
		if err != nil {
			return err
		}
		if removed {
			if err := s.appendExclusionClear(ctx, ref.Key); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) deleteExclusion(ctx context.Context, node string) (bool, error) {
	err := s.tx.QueryRowContext(ctx, `
DELETE FROM routing_exclusions
WHERE node_key = $1
RETURNING node_key`, node).Scan(new(string))
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) appendExclusionClear(ctx context.Context, key string) error {
	input, err := exclusionClearEvent(
		key,
		journal.EventID(s.ids.Next()),
		time.Now().UTC(),
	)
	if err != nil {
		return err
	}
	_, err = eventstore.New(s.tx).Append(ctx, input)
	return err
}

func (s *Store) remove(ctx context.Context, ref declaration.RecordRef) error {
	switch ref.Kind {
	case "route":
		return s.removeRoute(ctx, ref.Key)
	case "node":
		return s.removeNode(ctx, ref.Key)
	case "need":
		return s.removeNeed(ctx, ref.Key)
	case "item":
		return s.removeItem(ctx, ref.Key)
	case "schema":
		return s.removeSchema(ctx, ref.Key)
	default:
		return nil
	}
}

func (s *Store) removeRoute(ctx context.Context, key string) error {
	need, order, ok := strings.Cut(key, "/")
	if !ok {
		return declaration.ErrUnsafeDelta
	}
	value, err := strconv.Atoi(order)
	if err != nil {
		return declaration.ErrUnsafeDelta
	}
	_, err = s.tx.ExecContext(ctx, `
DELETE FROM routing_rules
WHERE need_kind_key = $1
  AND rule_order = $2`, need, value)
	return err
}

func (s *Store) removeNode(ctx context.Context, key string) error {
	if err := s.requireNoLiveChannel(ctx, key); err != nil {
		return err
	}
	removed, err := s.deleteExclusion(ctx, key)
	if err != nil {
		return err
	}
	if removed {
		if err := s.appendExclusionClear(ctx, key); err != nil {
			return err
		}
	}
	for _, statement := range []string{
		`DELETE FROM node_capabilities WHERE node_key = $1`,
		`DELETE FROM channels WHERE node_key = $1`,
		`DELETE FROM nodes WHERE key = $1`,
	} {
		if _, err := s.tx.ExecContext(ctx, statement, key); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) requireNoLiveChannel(ctx context.Context, node string) error {
	var live int
	err := s.tx.QueryRowContext(ctx, `
SELECT count(*)
FROM channels c
WHERE c.node_key = $1
  AND (
    EXISTS (SELECT 1 FROM channel_entries e WHERE e.channel_key = c.key)
    OR EXISTS (SELECT 1 FROM leases l WHERE l.channel_key = c.key)
  )`, node).Scan(&live)
	if err != nil {
		return err
	}
	if live > 0 {
		return declaration.ErrUnsafeDelta
	}
	return nil
}

func (s *Store) removeNeed(ctx context.Context, key string) error {
	_, err := s.tx.ExecContext(ctx, `DELETE FROM need_kinds WHERE key = $1`, key)
	return err
}

func (s *Store) removeItem(ctx context.Context, key string) error {
	_, err := s.tx.ExecContext(ctx, `DELETE FROM item_kinds WHERE key = $1`, key)
	return err
}

func (s *Store) removeSchema(ctx context.Context, key string) error {
	_, err := s.tx.ExecContext(ctx, `DELETE FROM schema_documents WHERE key = $1`, key)
	return err
}

func (s *Store) Commit() error {
	return s.tx.Commit()
}

func (s *Store) Rollback() error {
	return s.tx.Rollback()
}

func Open(ctx context.Context, db *sql.DB, searchPath string) (*Store, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if err := postgres.SetSearchPath(ctx, tx, searchPath); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return &Store{tx: tx, ids: ids.Random{}}, nil
}

func exclusionClearEvent(key string, id journal.EventID, appendedAt time.Time) (journal.EventInput, error) {
	node := registry.NodeKey(key)
	body, err := payloads.ExclusionClear(node)
	if err != nil {
		return journal.EventInput{}, err
	}
	return journal.EventInput{
		ID:         id,
		Coordinate: journal.NodeCoordinate(node),
		Kind:       payloads.ExclusionClearKind,
		AppendedAt: appendedAt,
		Payload:    body,
	}, nil
}
