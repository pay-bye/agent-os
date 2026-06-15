package leases

import (
	"errors"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/channel"
)

func TestExtendBuildsDigestAndClockFacts(t *testing.T) {
	command, err := Extend(ExtendInput{
		Lease:     "x16",
		Token:     "x-token",
		ExpiresAt: instant(10),
	}, instant(0))
	if err != nil {
		t.Fatal(err)
	}

	if command.CheckedAt != instant(0) {
		t.Fatalf("checked at = %s, want %s", command.CheckedAt, instant(0))
	}
	if command.ExpiresAt != instant(10) {
		t.Fatalf("expires at = %s, want %s", command.ExpiresAt, instant(10))
	}
	if command.TokenDigest != channel.Digest("GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14") {
		t.Fatalf("token digest = %q", command.TokenDigest)
	}
}

func TestHeartbeatBuildsDigestAndClockFacts(t *testing.T) {
	command, err := Heartbeat(HeartbeatInput{Lease: "x16", Token: "x-token"}, instant(0))
	if err != nil {
		t.Fatal(err)
	}

	if command.CheckedAt != instant(0) {
		t.Fatalf("checked at = %s, want %s", command.CheckedAt, instant(0))
	}
	if command.TokenDigest != channel.Digest("GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14") {
		t.Fatalf("token digest = %q", command.TokenDigest)
	}
}

func TestCommandsRejectBlankToken(t *testing.T) {
	if _, err := Extend(ExtendInput{Token: " "}, instant(0)); !errors.Is(err, channel.ErrEmptyToken) {
		t.Fatalf("extend error = %v, want empty token", err)
	}
	if _, err := Heartbeat(HeartbeatInput{Token: " "}, instant(0)); !errors.Is(err, channel.ErrEmptyToken) {
		t.Fatalf("heartbeat error = %v, want empty token", err)
	}
}

func instant(offset int) time.Time {
	return time.Date(2026, 5, 18, 12, offset, 0, 0, time.UTC)
}
