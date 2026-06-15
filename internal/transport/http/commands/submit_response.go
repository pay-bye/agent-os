package commands

import (
	"github.com/pay-bye/agent-os/internal/kernel"
)

type submitRoutedResponse struct {
	WorkItem string `json:"work_item_id"`
	Routed   bool   `json:"routed"`
	Channel  string `json:"channel_key"`
}

type submitUnroutedResponse struct {
	WorkItem string `json:"work_item_id"`
	Routed   bool   `json:"routed"`
}

func submitBody(result kernel.SubmitResult) any {
	if result.Routed {
		return submitRoutedResponse{
			WorkItem: result.WorkItem.String(),
			Routed:   true,
			Channel:  result.Channel.String(),
		}
	}
	return submitUnroutedResponse{WorkItem: result.WorkItem.String(), Routed: false}
}
