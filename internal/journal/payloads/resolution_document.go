package payloads

import "github.com/pay-bye/agent-os/internal/channel"

type Resolution struct {
	Lease          channel.Lease
	FailurePayload []byte
}

func marshalResolution(resolution Resolution) ([]byte, error) {
	payload := map[string]any{
		"lease_id":     resolution.Lease.ID().String(),
		"work_item_id": resolution.Lease.WorkItem().String(),
	}
	if len(resolution.FailurePayload) > 0 {
		payload["failure_payload"] = raw(resolution.FailurePayload)
	}
	return marshal(payload)
}
