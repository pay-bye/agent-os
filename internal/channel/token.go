package channel

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
)

var ErrEmptyToken = errors.New("lease token is empty")

type Token string

func NewToken(value string) (Token, error) {
	if strings.TrimSpace(value) == "" {
		return "", ErrEmptyToken
	}
	return Token(value), nil
}

func (t Token) String() string {
	return string(t)
}

func (t Token) Digest() Digest {
	sum := sha256.Sum256([]byte(t.String()))
	return Digest(base64.RawURLEncoding.EncodeToString(sum[:]))
}

type Digest string

func (d Digest) String() string {
	return string(d)
}

func (d Digest) Matches(other Digest) bool {
	left, err := decodeDigest(d)
	if err != nil {
		return false
	}
	right, err := decodeDigest(other)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(left, right) == 1
}

func DigestFor(token Token) (Digest, error) {
	validated, err := NewToken(token.String())
	if err != nil {
		return "", err
	}
	return validated.Digest(), nil
}

func decodeDigest(digest Digest) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(digest.String())
}
