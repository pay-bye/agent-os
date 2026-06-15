package http

import (
	"errors"
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/registry"
)

func TestClassifiesKnownCommandErrors(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		code  int
		token string
	}{
		{name: "unknown vocabulary", err: registry.ChannelNotFound("x15"), code: 404, token: "unknown_vocabulary"},
		{name: "empty queue", err: channel.ErrEmpty, code: 404, token: "empty_queue"},
		{name: "invalid lease", err: channel.ErrInvalidLease, code: 404, token: "invalid_lease"},
		{name: "expired lease", err: channel.ErrExpiredLease, code: 404, token: "expired_lease"},
		{name: "no route", err: registry.ErrNoRoute, code: 404, token: "no_route"},
		{name: "conflict", err: errors.New("write conflict"), code: 409, token: "conflict"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commands := &recordingCommands{submitErr: test.err}

			response := serve(t, commands, "POST", "/submit", `{
				"work_item_id": "x08",
				"item_kind": "x08",
				"payload": {},
				"declared_needs": []
			}`)

			requireCode(t, response, test.code)
			requireJSONContent(t, response)
			requireBody(t, response, map[string]any{"error": test.token})
		})
	}
}
