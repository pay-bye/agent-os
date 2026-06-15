package commands

import (
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestClaimWritesEmptyResponse(t *testing.T) {
	commands := &recordingCommands{claimResult: kernel.ClaimResult{Empty: true}}

	response := serve(t, commands, "POST", "/claim", `{
		"channel_key": "x15",
		"lease_id": "x16",
		"lease_seconds": 600
	}`)

	requireCode(t, response, 200)
	requireJSONContent(t, response)
	requireCalls(t, commands.calls, "claim")
	requireBody(t, response, map[string]any{"empty": true})
}

func TestClaimWritesLeaseResponse(t *testing.T) {
	lease := mustLease(t, "x16", instant(10))
	commands := &recordingCommands{
		claimResult: kernel.ClaimResult{
			Empty:    false,
			Lease:    lease,
			Token:    channel.Token("x-token"),
			WorkItem: workitem.ID("x08"),
			Payload:  []byte(`{"value":"x75"}`),
		},
	}

	response := serve(t, commands, "POST", "/claim", `{
		"channel_key": "x15",
		"lease_id": "x16",
		"lease_seconds": 600
	}`)

	requireCode(t, response, 200)
	requireJSONContent(t, response)
	requireCalls(t, commands.calls, "claim")
	requireBody(t, response, map[string]any{
		"empty":        false,
		"lease_id":     "x16",
		"lease_token":  "x-token",
		"work_item_id": "x08",
		"payload":      map[string]any{"value": "x75"},
		"expires_at":   "2026-05-18T12:10:00Z",
	})
}

func TestClaimRejectsNonPositiveLeaseDuration(t *testing.T) {
	commands := &recordingCommands{}

	response := serve(t, commands, "POST", "/claim", `{
		"channel_key": "x15",
		"lease_id": "x16",
		"lease_seconds": 0
	}`)

	requireCode(t, response, 400)
	requireCalls(t, commands.calls)
	requireBody(t, response, map[string]any{"error": "invalid_input"})
}
