package config

import (
	"github.com/pay-bye/agent-os/internal/declaration"
	"time"
)

const DefaultDeclaration = declaration.DefaultPath

func defaultGrace() time.Duration {
	return 10 * time.Second
}

func choose(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func chooseDuration(values ...time.Duration) time.Duration {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func seconds(value int) time.Duration {
	if value <= 0 {
		return 0
	}
	return time.Duration(value) * time.Second
}
