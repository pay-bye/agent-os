package claiming

import (
	"errors"
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/registry"
	"time"
)

var ErrInvalidLeaseDuration = errors.New("lease duration must be positive")

type Input struct {
	Channel       registry.ChannelKey
	Lease         channel.LeaseID
	LeaseDuration time.Duration
}

type Command struct {
	Channel   registry.ChannelKey
	Lease     channel.LeaseRequest
	ClaimedAt time.Time
}

type Tokens interface {
	Next() (channel.Token, error)
}

func New(input Input, now time.Time, tokens Tokens) (Command, channel.Token, error) {
	if input.LeaseDuration <= 0 {
		return Command{}, "", ErrInvalidLeaseDuration
	}
	token, err := tokens.Next()
	if err != nil {
		return Command{}, "", err
	}
	digest, err := channel.DigestFor(token)
	if err != nil {
		return Command{}, "", err
	}
	request := channel.LeaseRequest{
		ID:          input.Lease,
		TokenDigest: digest,
		GrantedAt:   now,
		ExpiresAt:   now.Add(input.LeaseDuration),
	}
	if err := request.Validate(); err != nil {
		return Command{}, "", err
	}
	return Command{
		Channel:   input.Channel,
		Lease:     request,
		ClaimedAt: now,
	}, token, nil
}
