package channel

import (
	"context"
	"database/sql"
	"errors"
	coordination "github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/registry"
	"time"
)

type commander interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Store struct {
	command commandFunc
	query   queryFunc
}

func (c *Store) Enqueue(ctx context.Context, input coordination.EntryInput) (coordination.Entry, error) {
	entry, err := coordination.NewEntry(input)
	if err != nil {
		return coordination.Entry{}, err
	}
	_, err = c.command(ctx, `
INSERT INTO channel_entries (id, channel_key, work_item_id, enqueued_at, available_at)
VALUES ($1, $2, $3, $4, $5)`,
		entry.ID().String(),
		entry.Channel().String(),
		entry.WorkItem().String(),
		entry.EnqueuedAt(),
		entry.AvailableAt(),
	)
	if err != nil {
		return coordination.Entry{}, err
	}
	return entry, nil
}

func (c *Store) Extend(
	ctx context.Context,
	id coordination.LeaseID,
	digest coordination.Digest,
	now time.Time,
	expiresAt time.Time,
) (coordination.Lease, error) {
	lease, err := c.Heartbeat(ctx, id, digest, now)
	if err != nil {
		return coordination.Lease{}, err
	}
	if !expiresAt.After(lease.ExpiresAt()) {
		return coordination.Lease{}, coordination.ErrNonIncreasingExtension
	}
	return c.writeExpiry(ctx, id, expiresAt)
}

func (c *Store) writeExpiry(
	ctx context.Context,
	id coordination.LeaseID,
	expiresAt time.Time,
) (coordination.Lease, error) {
	row := c.query(ctx, `
UPDATE leases
SET expires_at = $2
WHERE id = $1
RETURNING id, channel_entry_id, work_item_id, channel_key, granted_at, expires_at`,
		id.String(),
		expiresAt,
	)
	return LeaseFromRow(row)
}

func (c *Store) Heartbeat(
	ctx context.Context,
	id coordination.LeaseID,
	digest coordination.Digest,
	now time.Time,
) (coordination.Lease, error) {
	if id.String() == "" {
		return coordination.Lease{}, coordination.ErrEmptyLeaseID
	}
	record, err := c.findLease(ctx, id)
	if err != nil {
		return coordination.Lease{}, err
	}
	if !digest.Matches(record.digest) {
		return coordination.Lease{}, coordination.ErrInvalidLease
	}
	lease := record.lease
	if !lease.ExpiresAt().After(now) {
		return coordination.Lease{}, coordination.ErrExpiredLease
	}
	return lease, nil
}

func (c *Store) findLease(ctx context.Context, id coordination.LeaseID) (leaseRecord, error) {
	row := c.query(ctx, `
SELECT id, channel_entry_id, work_item_id, channel_key, granted_at, expires_at, token_digest
FROM leases
WHERE id = $1
FOR UPDATE`, id.String())
	record, err := leaseRecordFromRow(row)
	if errors.Is(err, coordination.ErrEmpty) {
		return leaseRecord{}, coordination.ErrInvalidLease
	}
	return record, err
}

func (c *Store) Dequeue(
	ctx context.Context,
	key registry.ChannelKey,
	request coordination.LeaseRequest,
) (coordination.Lease, error) {
	if err := request.Validate(); err != nil {
		return coordination.Lease{}, err
	}
	row := c.query(ctx, `
WITH removed AS (
  DELETE FROM leases WHERE expires_at <= $2
),
candidate AS (
  SELECT e.id, e.work_item_id, e.channel_key
  FROM channel_entries e
  WHERE e.channel_key = $1
    AND e.available_at <= $2
    AND NOT EXISTS (
      SELECT 1 FROM leases l WHERE l.channel_entry_id = e.id
    )
  ORDER BY e.available_at, e.enqueued_at, e.id
  LIMIT 1
),
granted AS (
  INSERT INTO leases (id, channel_entry_id, work_item_id, channel_key, granted_at, expires_at, token_digest)
  SELECT $3, candidate.id, candidate.work_item_id, candidate.channel_key, $2, $4, $5
  FROM candidate
  ON CONFLICT (channel_entry_id) DO NOTHING
  RETURNING id, channel_entry_id, work_item_id, channel_key, granted_at, expires_at
)
SELECT id, channel_entry_id, work_item_id, channel_key, granted_at, expires_at FROM granted`,
		key.String(),
		request.GrantedAt,
		request.ID.String(),
		request.ExpiresAt,
		request.TokenDigest.String(),
	)
	return LeaseFromRow(row)
}

func (c *Store) PrepareAck(
	ctx context.Context,
	id coordination.LeaseID,
	now time.Time,
) (coordination.Preparation, error) {
	return c.prepare(ctx, coordination.PreparationInput{Lease: id, Kind: coordination.Ack}, now)
}

func (c *Store) prepare(
	ctx context.Context,
	input coordination.PreparationInput,
	now time.Time,
) (coordination.Preparation, error) {
	item, err := coordination.NewPreparation(input)
	if err != nil {
		return coordination.Preparation{}, err
	}
	var expiresAt sql.NullTime
	err = c.query(ctx, `SELECT expires_at FROM leases WHERE id = $1`, item.Lease().String()).Scan(&expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return coordination.Preparation{}, coordination.ErrInvalidLease
	}
	if err != nil {
		return coordination.Preparation{}, err
	}
	if !expiresAt.Valid || !expiresAt.Time.After(now) {
		return coordination.Preparation{}, coordination.ErrExpiredLease
	}
	return item, nil
}

func (c *Store) PrepareNack(
	ctx context.Context,
	id coordination.LeaseID,
	now time.Time,
) (coordination.Preparation, error) {
	return c.prepare(ctx, coordination.PreparationInput{Lease: id, Kind: coordination.Nack}, now)
}

func New(db commander) *Store {
	return &Store{
		command: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			return db.ExecContext(ctx, query, args...)
		},
		query: func(ctx context.Context, query string, args ...any) rowScanner {
			return db.QueryRowContext(ctx, query, args...)
		},
	}
}
