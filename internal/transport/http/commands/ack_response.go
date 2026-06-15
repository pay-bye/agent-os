package commands

import (
	"github.com/pay-bye/agent-os/internal/kernel"
)

type ackRoutedResponse struct {
	Resolved bool   `json:"resolved"`
	Routed   bool   `json:"routed"`
	Channel  string `json:"channel_key"`
}

type ackUnroutedResponse struct {
	Resolved bool `json:"resolved"`
	Routed   bool `json:"routed"`
}

func ackBody(result kernel.ResolutionResult) any {
	if result.Routed {
		return ackRoutedResponse{
			Resolved: result.Resolved,
			Routed:   true,
			Channel:  result.Channel.String(),
		}
	}
	return ackUnroutedResponse{Resolved: result.Resolved, Routed: false}
}
