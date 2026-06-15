package kernel

import (
	"encoding/base64"
	"testing"
)

func TestSecureTokensReturnsEncodedEntropy(t *testing.T) {
	token, err := secureTokens{}.Next()
	if err != nil {
		t.Fatal(err)
	}

	entropy, err := base64.RawURLEncoding.DecodeString(token.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(entropy) != tokenBytes {
		t.Fatalf("entropy length = %d, want %d", len(entropy), tokenBytes)
	}
}
