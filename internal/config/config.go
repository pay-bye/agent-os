package config

import (
	"time"
)

type Input struct {
	File            string
	DatabaseURL     string
	Listen          string
	Declaration     string
	Grace           time.Duration
	Env             Env
	RequireDatabase bool
	RequireListen   bool
}

type Values struct {
	DatabaseURL string
	Listen      string
	Declaration string
	Grace       time.Duration
}
