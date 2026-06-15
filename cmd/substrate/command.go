package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/pay-bye/agent-os/internal/config"
	"github.com/pay-bye/agent-os/internal/declaration"
	"github.com/pay-bye/agent-os/internal/declaration/execution"
	"github.com/pay-bye/agent-os/internal/processlog"
	"github.com/pay-bye/agent-os/internal/transport/http/security/credential"
	"io"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type calls interface {
	Serve(context.Context, serverInput) error
	GenerateCredential() (credential.GeneratedCredential, error)
	Init(declaration.InitInput) error
	Preview(context.Context, execution.Input) (declaration.Delta, error)
	Apply(context.Context, execution.Input) (declaration.Delta, error)
}

func run(ctx context.Context, args []string, out io.Writer, errOut io.Writer, calls calls) int {
	recorder := processlog.NewSink(errOut, clock{})
	if len(args) == 0 {
		fmt.Fprintln(errOut, help())
		return 2
	}
	code, err := dispatch(ctx, args, out, errOut, calls, recorder)
	if err != nil {
		fmt.Fprintln(errOut, err)
	}
	return code
}

func dispatch(
	ctx context.Context,
	args []string,
	out io.Writer,
	errOut io.Writer,
	calls calls,
	recorder processlog.Recorder,
) (int, error) {
	if topHelpRequested(args) {
		fmt.Fprintln(out, help())
		return 0, nil
	}
	switch args[0] {
	case "serve":
		return executeServe(ctx, args[1:], out, errOut, calls, recorder)
	case "credential":
		return executeCredential(args[1:], out, calls)
	case "init":
		return executeInit(args[1:], out, errOut, calls)
	case "preview":
		return executePreview(ctx, args[1:], out, errOut, calls, recorder)
	case "apply":
		return executeApply(ctx, args[1:], out, errOut, calls, recorder)
	default:
		return 2, fmt.Errorf("unknown command: %s", args[0])
	}
}

func executeServe(
	ctx context.Context,
	args []string,
	out io.Writer,
	errOut io.Writer,
	calls calls,
	recorder processlog.Recorder,
) (int, error) {
	if helpRequested(args) {
		fmt.Fprintln(out, serveHelp())
		return 0, nil
	}
	flags, err := parseProcessFlags("serve", args, errOut)
	if err != nil {
		return 2, err
	}
	config, err := processConfig(flags, environment(calls), true, true, recorder)
	if err != nil {
		return 1, redactError(flags, err)
	}
	verifier, err := credential.LoadVerifier(credential.VerifierInput{
		Digests: flags.verifierDigests,
		File:    flags.verifierFile,
	})
	if err != nil {
		return 1, redactError(flags, err)
	}
	var operatorVerifier credential.OperatorKeyVerifier
	if flags.operatorFile != "" {
		verifier, err := credential.LoadOperatorVerifier(credential.OperatorVerifierInput{File: flags.operatorFile})
		if err != nil {
			return 1, redactError(flags, err)
		}
		operatorVerifier = verifier
	}
	input := serverInput{
		config:   config,
		recorder: recorder,
		verifier: verifier,
		operator: operatorVerifier,
	}
	if err := calls.Serve(ctx, input); err != nil {
		return 1, redactError(flags, err)
	}
	return 0, nil
}

func processConfig(
	flags processFlags,
	env config.Env,
	requireDatabase bool,
	requireListen bool,
	recorder processlog.Recorder,
) (config.Values, error) {
	values, err := configFromFlags(flags, env, requireDatabase, requireListen)
	if err != nil {
		record(recorder, processlog.ConfigValidated(processlog.Failed, processlog.ConfigInvalid))
		return config.Values{}, err
	}
	record(recorder, processlog.ConfigValidated(processlog.Succeeded, ""))
	return values, nil
}

func record(recorder processlog.Recorder, item processlog.Record) {
	if recorder != nil {
		recorder.Record(item)
	}
}

func executeCredential(args []string, out io.Writer, calls calls) (int, error) {
	if len(args) == 1 && helpRequested(args) {
		fmt.Fprintln(out, credentialHelp())
		return 0, nil
	}
	if len(args) == 2 && args[0] == "generate" && helpRequested(args[1:]) {
		fmt.Fprintln(out, credentialGenerateHelp())
		return 0, nil
	}
	if len(args) != 1 || args[0] != "generate" {
		return 2, fmt.Errorf("unknown credential command")
	}
	credential, err := calls.GenerateCredential()
	if err != nil {
		return 1, err
	}
	if err := json.NewEncoder(out).Encode(credential); err != nil {
		return 1, err
	}
	return 0, nil
}

func executeInit(args []string, out io.Writer, errOut io.Writer, calls calls) (int, error) {
	if helpRequested(args) {
		fmt.Fprintln(out, initHelp())
		return 0, nil
	}
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(errOut)
	path := flags.String("from", declaration.DefaultPath, "")
	yes := flags.Bool("yes", false, "")
	if err := flags.Parse(args); err != nil {
		return 2, err
	}
	if flags.NArg() != 0 {
		return 2, fmt.Errorf("unexpected argument: %s", flags.Arg(0))
	}
	if err := calls.Init(declaration.InitInput{Path: *path, Yes: *yes}); err != nil {
		return 1, err
	}
	return 0, nil
}

func helpRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func topHelpRequested(args []string) bool {
	return len(args) == 1 && helpRequested(args)
}

func environment(calls calls) config.Env {
	env, ok := calls.(config.Env)
	if !ok {
		return nil
	}
	return env
}
