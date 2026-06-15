package main

import (
	"flag"
	"fmt"
	"github.com/pay-bye/agent-os/internal/config"
	"github.com/pay-bye/agent-os/internal/declaration/execution"
	"github.com/pay-bye/agent-os/internal/processlog"
	"io"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type processFlags struct {
	configPath      string
	databaseURL     string
	listen          string
	declaration     string
	grace           time.Duration
	verifierDigests repeatedValues
	verifierFile    string
	operatorFile    string
}

type runtimeConfig struct {
	values map[string]string
}

func (c runtimeConfig) LookupEnv(key string) (string, bool) {
	value, ok := c.values[key]
	return value, ok
}

type repeatedValues []string

func (v *repeatedValues) String() string {
	return fmt.Sprint([]string(*v))
}

func (v *repeatedValues) Set(value string) error {
	*v = append(*v, value)
	return nil
}

func configFromFlags(
	values processFlags,
	env config.Env,
	requireDatabase bool,
	requireListen bool,
) (config.Values, error) {
	return config.Load(config.Input{
		File:            values.configPath,
		DatabaseURL:     values.databaseURL,
		Listen:          values.listen,
		Declaration:     values.declaration,
		Grace:           values.grace,
		Env:             env,
		RequireDatabase: requireDatabase,
		RequireListen:   requireListen,
	})
}

func inputFromFlags(
	values processFlags,
	env config.Env,
	storage execution.Storage,
	recorder processlog.Recorder,
) (execution.Input, error) {
	resolved, err := processConfig(values, env, true, false, recorder)
	if err != nil {
		return execution.Input{}, err
	}
	return execution.Input{
		DatabaseURL: resolved.DatabaseURL,
		Declaration: resolved.Declaration,
		Recorder:    recorder,
		Storage:     storage,
	}, nil
}

func redactError(values processFlags, err error) error {
	return fmt.Errorf("%s", redact(err.Error(), secrets(values)))
}

func secrets(values processFlags) []string {
	items := []string{values.databaseURL, values.verifierFile, values.operatorFile}
	items = append(items, values.verifierDigests...)
	return items
}

func loadConfig(env []string) runtimeConfig {
	values := map[string]string{}
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			values[key] = value
		}
	}
	return runtimeConfig{values: values}
}

func parseProcessFlags(name string, args []string, output io.Writer) (processFlags, error) {
	values := processFlags{}
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(output)
	flags.StringVar(&values.configPath, "config", "", "")
	flags.StringVar(&values.databaseURL, "database-url", "", "")
	flags.StringVar(&values.listen, "listen", "", "")
	flags.StringVar(&values.declaration, "from", "", "")
	flags.DurationVar(&values.grace, "shutdown-grace", 0, "")
	flags.Var(&values.verifierDigests, "verifier-digest", "")
	flags.StringVar(&values.verifierFile, "verifier-file", "", "")
	flags.StringVar(&values.operatorFile, "operator-verifier-file", "", "")
	if err := flags.Parse(args); err != nil {
		return processFlags{}, err
	}
	if flags.NArg() != 0 {
		return processFlags{}, fmt.Errorf("unexpected argument: %s", flags.Arg(0))
	}
	return values, nil
}
