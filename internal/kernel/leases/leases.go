package leases

import (
	"github.com/pay-bye/agent-os/internal/channel"
	"time"
)

type ExtendInput struct {
	Lease     channel.LeaseID
	Token     channel.Token
	ExpiresAt time.Time
}

type ExtendCommand struct {
	Lease       channel.LeaseID
	TokenDigest channel.Digest
	CheckedAt   time.Time
	ExpiresAt   time.Time
}

type HeartbeatInput struct {
	Lease channel.LeaseID
	Token channel.Token
}

type HeartbeatCommand struct {
	Lease       channel.LeaseID
	TokenDigest channel.Digest
	CheckedAt   time.Time
}

func Extend(input ExtendInput, now time.Time) (ExtendCommand, error) {
	digest, err := channel.DigestFor(input.Token)
	if err != nil {
		return ExtendCommand{}, err
	}
	return ExtendCommand{
		Lease:       input.Lease,
		TokenDigest: digest,
		CheckedAt:   now,
		ExpiresAt:   input.ExpiresAt,
	}, nil
}

func Heartbeat(input HeartbeatInput, now time.Time) (HeartbeatCommand, error) {
	digest, err := channel.DigestFor(input.Token)
	if err != nil {
		return HeartbeatCommand{}, err
	}
	return HeartbeatCommand{
		Lease:       input.Lease,
		TokenDigest: digest,
		CheckedAt:   now,
	}, nil
}
