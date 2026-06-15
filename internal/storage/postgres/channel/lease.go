package channel

import (
	"database/sql"
	"errors"
	coordination "github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
	"time"
)

type leaseInput struct {
	id          string
	entry       string
	workItem    string
	channel     string
	grantedAt   time.Time
	expiresAt   time.Time
	tokenDigest string
}

type leaseRecord struct {
	lease  coordination.Lease
	digest coordination.Digest
}

func LeaseFromRow(row rowScanner) (coordination.Lease, error) {
	var input leaseInput
	if err := row.Scan(
		&input.id,
		&input.entry,
		&input.workItem,
		&input.channel,
		&input.grantedAt,
		&input.expiresAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return coordination.Lease{}, coordination.ErrEmpty
		}
		return coordination.Lease{}, err
	}
	return newLease(input)
}

func leaseRecordFromRow(row rowScanner) (leaseRecord, error) {
	var input leaseInput
	if err := row.Scan(
		&input.id,
		&input.entry,
		&input.workItem,
		&input.channel,
		&input.grantedAt,
		&input.expiresAt,
		&input.tokenDigest,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return leaseRecord{}, coordination.ErrEmpty
		}
		return leaseRecord{}, err
	}
	lease, err := newLease(input)
	if err != nil {
		return leaseRecord{}, err
	}
	return leaseRecord{lease: lease, digest: coordination.Digest(input.tokenDigest)}, nil
}

func newLease(input leaseInput) (coordination.Lease, error) {
	return coordination.NewLease(coordination.LeaseInput{
		ID:        coordination.LeaseID(input.id),
		Entry:     coordination.EntryID(input.entry),
		Channel:   registry.ChannelKey(input.channel),
		WorkItem:  workitem.ID(input.workItem),
		GrantedAt: input.grantedAt,
		ExpiresAt: input.expiresAt,
	})
}
