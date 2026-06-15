package config

import (
	"testing"
	"time"
)

func TestLoadUsesProcessDefaults(t *testing.T) {
	config, err := Load(Input{Env: mapEnv{}})
	if err != nil {
		t.Fatal(err)
	}

	if config.Declaration != DefaultDeclaration {
		t.Fatalf("declaration = %q", config.Declaration)
	}
	if config.Grace != 10*time.Second {
		t.Fatalf("grace = %s", config.Grace)
	}
}
