package commands

import (
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
	nethttp "net/http"
	"time"
)

type extendRequest struct {
	Lease              string `json:"lease_id"`
	Token              string `json:"lease_token"`
	RequestedExpiresAt string `json:"requested_expires_at"`
}

func decodeExtend(request *nethttp.Request) (kernel.ExtendInput, error) {
	var body extendRequest
	if err := codec.DecodeBody(request, &body); err != nil {
		return kernel.ExtendInput{}, err
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, body.RequestedExpiresAt)
	token, tokenErr := channel.NewToken(body.Token)
	if err != nil || codec.Blank(body.Lease) || tokenErr != nil {
		return kernel.ExtendInput{}, codec.ErrInvalidInput
	}
	return kernel.ExtendInput{
		Lease:     channel.LeaseID(body.Lease),
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}
