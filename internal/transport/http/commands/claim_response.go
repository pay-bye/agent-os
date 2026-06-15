package commands

import (
	"encoding/json"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
)

type claimEmptyResponse struct {
	Empty bool `json:"empty"`
}

type claimLeaseResponse struct {
	Empty     bool            `json:"empty"`
	LeaseID   string          `json:"lease_id"`
	Token     string          `json:"lease_token"`
	WorkItem  string          `json:"work_item_id"`
	Payload   json.RawMessage `json:"payload"`
	ExpiresAt string          `json:"expires_at"`
}

func claimBody(result kernel.ClaimResult) (any, error) {
	if result.Empty {
		return claimEmptyResponse{Empty: true}, nil
	}
	payload, err := codec.ResponsePayload(result.Payload)
	if err != nil {
		return nil, err
	}
	return claimLeaseResponse{
		Empty:     false,
		LeaseID:   result.Lease.ID().String(),
		Token:     result.Token.String(),
		WorkItem:  result.WorkItem.String(),
		Payload:   payload,
		ExpiresAt: codec.FormatTime(result.Lease.ExpiresAt()),
	}, nil
}
