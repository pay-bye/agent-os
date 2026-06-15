package commands

import (
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
	nethttp "net/http"
	"time"
)

type claimRequest struct {
	Channel      string `json:"channel_key"`
	Lease        string `json:"lease_id"`
	LeaseSeconds int    `json:"lease_seconds"`
}

func decodeClaim(request *nethttp.Request) (kernel.ClaimInput, error) {
	var body claimRequest
	if err := codec.DecodeBody(request, &body); err != nil {
		return kernel.ClaimInput{}, err
	}
	if codec.Blank(body.Channel) || codec.Blank(body.Lease) || body.LeaseSeconds <= 0 {
		return kernel.ClaimInput{}, codec.ErrInvalidInput
	}
	return kernel.ClaimInput{
		Channel:       registry.ChannelKey(body.Channel),
		Lease:         channel.LeaseID(body.Lease),
		LeaseDuration: time.Duration(body.LeaseSeconds) * time.Second,
	}, nil
}
