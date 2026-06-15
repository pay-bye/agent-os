package main

import (
	"context"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	config := loadConfig(os.Environ())
	runtime := newRuntime(config)
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, runtime))
}
