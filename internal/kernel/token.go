package kernel

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/pay-bye/agent-os/internal/channel"
)

const tokenBytes = 32

type secureTokens struct{}

func (secureTokens) Next() (channel.Token, error) {
	value := make([]byte, tokenBytes)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return channel.NewToken(base64.RawURLEncoding.EncodeToString(value))
}
