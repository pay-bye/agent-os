package claiming

import (
	"errors"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/registry"
)

func TestNewRejectsInvalidDuration(t *testing.T) {
	_, _, err := New(Input{Lease: "x16"}, instant(0), fixedTokens{})

	if !errors.Is(err, ErrInvalidLeaseDuration) {
		t.Fatalf("error = %v, want invalid duration", err)
	}
}

func TestNewStopsBeforeLeaseRequestWhenTokenGenerationFails(t *testing.T) {
	want := errors.New("token unavailable")

	_, _, err := New(Input{
		Channel:       "x15",
		Lease:         "x16",
		LeaseDuration: time.Minute,
	}, instant(0), fixedTokens{err: want})

	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want token failure", err)
	}
}

func TestNewStoresDigestAndReturnsToken(t *testing.T) {
	command, token, err := New(Input{
		Channel:       registry.ChannelKey("x15"),
		Lease:         channel.LeaseID("x16"),
		LeaseDuration: 10 * time.Minute,
	}, instant(0), fixedTokens{value: "x-token"})
	if err != nil {
		t.Fatal(err)
	}

	if command.Lease.GrantedAt != instant(0) {
		t.Fatalf("granted at = %s, want %s", command.Lease.GrantedAt, instant(0))
	}
	if command.Lease.TokenDigest != channel.Digest("GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14") {
		t.Fatalf("lease token digest = %q", command.Lease.TokenDigest)
	}
	if !command.Lease.ExpiresAt.Equal(instant(0).Add(10 * time.Minute)) {
		t.Fatalf("expires at = %s", command.Lease.ExpiresAt)
	}
	if token != channel.Token("x-token") {
		t.Fatalf("token = %q, want x-token", token)
	}
}

type fixedTokens struct {
	value channel.Token
	err   error
}

func (t fixedTokens) Next() (channel.Token, error) {
	return t.value, t.err
}

func instant(offset int) time.Time {
	return time.Date(2026, 5, 18, 12, offset, 0, 0, time.UTC)
}
