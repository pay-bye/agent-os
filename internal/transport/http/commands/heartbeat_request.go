package commands

import (
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
	nethttp "net/http"
)

type heartbeatRequest struct {
	Lease string `json:"lease_id"`
	Token string `json:"lease_token"`
}

func decodeHeartbeat(request *nethttp.Request) (kernel.HeartbeatInput, error) {
	var body heartbeatRequest
	if err := codec.DecodeBody(request, &body); err != nil {
		return kernel.HeartbeatInput{}, err
	}
	token, err := channel.NewToken(body.Token)
	if codec.Blank(body.Lease) || err != nil {
		return kernel.HeartbeatInput{}, codec.ErrInvalidInput
	}
	return kernel.HeartbeatInput{Lease: channel.LeaseID(body.Lease), Token: token}, nil
}
