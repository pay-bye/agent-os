package ids

import (
	"crypto/rand"
	"encoding/base64"
	"io"
)

type Random struct{}

func (Random) Next() string {
	value, err := New()
	if err != nil {
		panic(err)
	}
	return value
}

func New() (string, error) {
	return newFrom(rand.Reader)
}

func newFrom(reader io.Reader) (string, error) {
	raw := make([]byte, 16)
	if _, err := io.ReadFull(reader, raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
