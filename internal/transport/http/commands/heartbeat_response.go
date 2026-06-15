package commands

import (
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
)

type heartbeatResponse struct {
	LeaseID   string `json:"lease_id"`
	ExpiresAt string `json:"expires_at"`
}

func heartbeatBody(lease channel.Lease) heartbeatResponse {
	return heartbeatResponse{
		LeaseID:   lease.ID().String(),
		ExpiresAt: codec.FormatTime(lease.ExpiresAt()),
	}
}
