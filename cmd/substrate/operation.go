package main

import (
	"context"
	"fmt"
	"github.com/pay-bye/agent-os/internal/config"
	"github.com/pay-bye/agent-os/internal/declaration"
	"github.com/pay-bye/agent-os/internal/declaration/execution"
	"github.com/pay-bye/agent-os/internal/processlog"
	"io"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type storageProvider interface {
	Storage() execution.Storage
}

func executePreview(
	ctx context.Context,
	args []string,
	out io.Writer,
	errOut io.Writer,
	calls calls,
	recorder processlog.Recorder,
) (int, error) {
	if helpRequested(args) {
		fmt.Fprintln(out, previewHelp())
		return 0, nil
	}
	flags, err := parseProcessFlags("preview", args, errOut)
	if err != nil {
		return 2, err
	}
	delta, err := callDelta(ctx, flags, environment(calls), calls, calls.Preview, recorder)
	if err != nil {
		return 1, redactError(flags, err)
	}
	if err := writeDelta(out, delta); err != nil {
		return 1, err
	}
	if !delta.Installable {
		return 1, nil
	}
	return 0, nil
}

func executeApply(
	ctx context.Context,
	args []string,
	out io.Writer,
	errOut io.Writer,
	calls calls,
	recorder processlog.Recorder,
) (int, error) {
	if helpRequested(args) {
		fmt.Fprintln(out, applyHelp())
		return 0, nil
	}
	flags, err := parseProcessFlags("apply", args, errOut)
	if err != nil {
		return 2, err
	}
	delta, err := callDelta(ctx, flags, environment(calls), calls, calls.Apply, recorder)
	if err != nil {
		return 1, redactError(flags, err)
	}
	if err := writeDelta(out, delta); err != nil {
		return 1, err
	}
	return 0, nil
}

func callDelta(
	ctx context.Context,
	flags processFlags,
	env config.Env,
	calls calls,
	call func(context.Context, execution.Input) (declaration.Delta, error),
	recorder processlog.Recorder,
) (declaration.Delta, error) {
	input, err := inputFromFlags(flags, env, storageFor(calls, recorder), recorder)
	if err != nil {
		return declaration.Delta{}, err
	}
	return call(ctx, input)
}

func storageFor(calls calls, recorder processlog.Recorder) execution.Storage {
	provider, ok := calls.(storageProvider)
	if !ok {
		return nil
	}
	item := provider.Storage()
	if _, ok := item.(storage); ok {
		return storage{recorder: recorder}
	}
	return item
}
