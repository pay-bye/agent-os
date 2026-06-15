package commands

import (
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
	nethttp "net/http"
)

type nackRequest struct {
	Lease          string              `json:"lease_id"`
	Token          string              `json:"lease_token"`
	DeclaredNeeds  []codec.NeedRequest `json:"declared_needs"`
	FailurePayload codec.Object        `json:"failure_payload"`
}

func decodeNack(request *nethttp.Request) (kernel.ResolutionInput, error) {
	var body nackRequest
	if err := codec.DecodeBody(request, &body); err != nil {
		return kernel.ResolutionInput{}, err
	}
	return codec.ResolutionInput(body.Lease, body.Token, body.DeclaredNeeds, codec.PayloadBytes(body.FailurePayload))
}
