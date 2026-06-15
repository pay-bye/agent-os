package channel

import (
	"errors"
	"testing"
)

func TestTokenDerivesDigest(t *testing.T) {
	token, err := NewToken("x-token")
	if err != nil {
		t.Fatal(err)
	}

	digest := token.Digest()

	if digest != Digest("GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14") {
		t.Fatalf("digest = %q", digest)
	}
}

func TestTokenRejectsBlankValue(t *testing.T) {
	_, err := NewToken(" ")

	if !errors.Is(err, ErrEmptyToken) {
		t.Fatalf("error = %v, want empty token", err)
	}
}

func TestDigestForValidatesTokenBeforeHashing(t *testing.T) {
	digest, err := DigestFor(Token("x-token"))
	if err != nil {
		t.Fatal(err)
	}

	if digest != Digest("GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14") {
		t.Fatalf("digest = %q", digest)
	}

	_, err = DigestFor(Token(" "))
	if !errors.Is(err, ErrEmptyToken) {
		t.Fatalf("error = %v, want empty token", err)
	}
}

func TestDigestMatchesEquivalentValue(t *testing.T) {
	token, err := NewToken("x-token")
	if err != nil {
		t.Fatal(err)
	}

	if !token.Digest().Matches(Digest("GAkyW6Zb2GqOpcgGYcs_HMHRYAZ9I3JVS4nCmAXqF14")) {
		t.Fatal("expected digest match")
	}
}

func TestDigestRejectsWrongOrMalformedValue(t *testing.T) {
	token, err := NewToken("x-token")
	if err != nil {
		t.Fatal(err)
	}

	for _, digest := range []Digest{"wrong", ""} {
		if token.Digest().Matches(digest) {
			t.Fatalf("digest %q matched", digest)
		}
	}
}
