package metrics

import (
	"context"
	"time"
)

type Aggregates struct {
	AvailableDepth int
	LeasesHeld     int
	LeasesExpired  int
}

type Store struct {
	query queryFunc
}

func (m *Store) Aggregates(ctx context.Context, now time.Time) (Aggregates, error) {
	var aggregates Aggregates
	err := m.query(ctx, `
SELECT
  (
    SELECT COUNT(*)
    FROM channel_entries e
    WHERE e.available_at <= $1
      AND NOT EXISTS (
        SELECT 1
        FROM leases l
        WHERE l.channel_entry_id = e.id
          AND l.expires_at > $1
      )
  ) AS queue_depth,
  (
    SELECT COUNT(*)
    FROM leases
    WHERE expires_at > $1
  ) AS leases_held,
  (
    SELECT COUNT(*)
    FROM leases
    WHERE expires_at <= $1
  ) AS leases_expired`,
		now,
	).Scan(&aggregates.AvailableDepth, &aggregates.LeasesHeld, &aggregates.LeasesExpired)
	return aggregates, err
}

func New(db rowReader) *Store {
	return &Store{
		query: func(ctx context.Context, query string, args ...any) rowScanner {
			return db.QueryRowContext(ctx, query, args...)
		},
	}
}
