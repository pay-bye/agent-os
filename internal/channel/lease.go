package channel

import (
	"errors"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
	"time"
)

var (
	ErrEmptyLeaseID           = errors.New("lease identity is empty")
	ErrEmptyDigest            = errors.New("lease token digest is empty")
	ErrMissingGrantedAt       = errors.New("lease granted time is missing")
	ErrMissingExpiresAt       = errors.New("lease expiry time is missing")
	ErrInvalidLeaseWindow     = errors.New("lease expiry must be after grant")
	ErrEmpty                  = errors.New("channel queue is empty")
	ErrInvalidLease           = errors.New("lease is invalid")
	ErrExpiredLease           = errors.New("lease is expired")
	ErrNonIncreasingExtension = errors.New("lease extension must increase expiry")
)

type LeaseID string

func (id LeaseID) String() string {
	return string(id)
}

type LeaseInput struct {
	ID        LeaseID
	Entry     EntryID
	Channel   registry.ChannelKey
	WorkItem  workitem.ID
	GrantedAt time.Time
	ExpiresAt time.Time
}

type LeaseRequest struct {
	ID          LeaseID
	TokenDigest Digest
	GrantedAt   time.Time
	ExpiresAt   time.Time
}

func (r LeaseRequest) Validate() error {
	if blank(r.ID.String()) {
		return ErrEmptyLeaseID
	}
	if blank(r.TokenDigest.String()) {
		return ErrEmptyDigest
	}
	if r.GrantedAt.IsZero() {
		return ErrMissingGrantedAt
	}
	if r.ExpiresAt.IsZero() {
		return ErrMissingExpiresAt
	}
	if !r.ExpiresAt.After(r.GrantedAt) {
		return ErrInvalidLeaseWindow
	}
	return nil
}

type Lease struct {
	id        LeaseID
	entry     EntryID
	channel   registry.ChannelKey
	workItem  workitem.ID
	grantedAt time.Time
	expiresAt time.Time
}

func NewLease(input LeaseInput) (Lease, error) {
	if err := validateLeaseInput(input); err != nil {
		return Lease{}, err
	}
	return Lease{
		id:        input.ID,
		entry:     input.Entry,
		channel:   input.Channel,
		workItem:  input.WorkItem,
		grantedAt: input.GrantedAt,
		expiresAt: input.ExpiresAt,
	}, nil
}

func (l Lease) ID() LeaseID {
	return l.id
}

func (l Lease) Entry() EntryID {
	return l.entry
}

func (l Lease) Channel() registry.ChannelKey {
	return l.channel
}

func (l Lease) WorkItem() workitem.ID {
	return l.workItem
}

func (l Lease) GrantedAt() time.Time {
	return l.grantedAt
}

func (l Lease) ExpiresAt() time.Time {
	return l.expiresAt
}

func validateLeaseInput(input LeaseInput) error {
	if blank(input.ID.String()) {
		return ErrEmptyLeaseID
	}
	if blank(input.Entry.String()) {
		return ErrEmptyEntryID
	}
	if blank(input.Channel.String()) {
		return ErrEmptyChannelKey
	}
	if blank(input.WorkItem.String()) {
		return ErrEmptyWorkItemID
	}
	if input.GrantedAt.IsZero() {
		return ErrMissingGrantedAt
	}
	if input.ExpiresAt.IsZero() {
		return ErrMissingExpiresAt
	}
	if !input.ExpiresAt.After(input.GrantedAt) {
		return ErrInvalidLeaseWindow
	}
	return nil
}
