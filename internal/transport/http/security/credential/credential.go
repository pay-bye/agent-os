package credential

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
)

var (
	ErrEmptyVerifier   = errors.New("verifier requires at least one digest")
	errInvalidVerifier = errors.New("verifier digest is invalid")
)

// Verifier holds SHA-256 digest material for bearer credential checks.
type Verifier struct {
	digests [][sha256.Size]byte
}

// NewVerifier builds a verifier from base64url SHA-256 digest strings.
func NewVerifier(digests ...string) (Verifier, error) {
	if len(digests) == 0 {
		return Verifier{}, ErrEmptyVerifier
	}
	seen := map[string]bool{}
	parsed := make([][sha256.Size]byte, 0, len(digests))
	for _, value := range digests {
		digest, key, err := parseDigest(value)
		if err != nil {
			return Verifier{}, err
		}
		if seen[key] {
			return Verifier{}, errInvalidVerifier
		}
		seen[key] = true
		parsed = append(parsed, digest)
	}
	return Verifier{digests: parsed}, nil
}

func (v Verifier) Accepts(credential string) bool {
	sum := sha256.Sum256([]byte(credential))
	matched := 0
	for _, digest := range v.digests {
		matched |= subtle.ConstantTimeCompare(sum[:], digest[:])
	}
	return matched == 1
}

func (v Verifier) Empty() bool {
	return len(v.digests) == 0
}

// GeneratedCredential carries a raw bearer credential and its verifier digest.
type GeneratedCredential struct {
	Credential     string `json:"credential"`
	VerifierDigest string `json:"verifier_digest"`
}

// GenerateCredential creates one raw bearer credential and its verifier digest.
func GenerateCredential() (GeneratedCredential, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return GeneratedCredential{}, err
	}
	credential := base64.RawURLEncoding.EncodeToString(raw)
	return GeneratedCredential{
		Credential:     credential,
		VerifierDigest: verifierDigest(credential),
	}, nil
}

func verifierDigest(credential string) string {
	sum := sha256.Sum256([]byte(credential))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func parseDigest(value string) ([sha256.Size]byte, string, error) {
	var digest [sha256.Size]byte
	if value == "" || strings.TrimSpace(value) != value || strings.Contains(value, "=") {
		return digest, "", errInvalidVerifier
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) != sha256.Size {
		return digest, "", errInvalidVerifier
	}
	key := base64.RawURLEncoding.EncodeToString(decoded)
	if key != value {
		return digest, "", errInvalidVerifier
	}
	copy(digest[:], decoded)
	return digest, key, nil
}
