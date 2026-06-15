package config

import (
	"os"
)

type Env interface {
	LookupEnv(string) (string, bool)
}

type osEnv struct{}

func (osEnv) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

func envDatabaseURL(env Env) string {
	if env == nil {
		env = osEnv{}
	}
	value, ok := env.LookupEnv("DATABASE_URL")
	if !ok {
		return ""
	}
	return value
}
