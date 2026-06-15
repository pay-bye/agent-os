package commands

import (
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
)

type extendResponse struct {
	LeaseID   string `json:"lease_id"`
	ExpiresAt string `json:"expires_at"`
}

func extendBody(lease channel.Lease) extendResponse {
	return extendResponse{
		LeaseID:   lease.ID().String(),
		ExpiresAt: codec.FormatTime(lease.ExpiresAt()),
	}
}
