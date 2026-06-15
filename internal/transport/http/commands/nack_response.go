package commands

import (
	"github.com/pay-bye/agent-os/internal/kernel"
)

type nackRoutedResponse struct {
	Resolved bool   `json:"resolved"`
	Routed   bool   `json:"routed"`
	Channel  string `json:"channel_key"`
}

type nackUnroutedResponse struct {
	Resolved bool `json:"resolved"`
	Routed   bool `json:"routed"`
}

func nackBody(result kernel.ResolutionResult) any {
	if result.Routed {
		return nackRoutedResponse{
			Resolved: result.Resolved,
			Routed:   true,
			Channel:  result.Channel.String(),
		}
	}
	return nackUnroutedResponse{Resolved: result.Resolved, Routed: false}
}
