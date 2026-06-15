package payloads

import (
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

const ItemSubmittedKind registry.JournalEventKindKey = "x40"

func ItemSubmitted(submission workitem.Submission) ([]byte, error) {
	return marshal(map[string]any{
		"work_item_id": submission.ID().String(),
		"item_kind":    submission.Kind().String(),
	})
}
