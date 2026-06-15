package execution

import (
	"context"
	"errors"
	"github.com/pay-bye/agent-os/internal/declaration"
	"github.com/pay-bye/agent-os/internal/processlog"
)

var errStorageRequired = errors.New("storage_required")

type Input struct {
	DatabaseURL string
	Declaration string
	Recorder    processlog.Recorder
	Storage     Storage
}

type Storage interface {
	Open(context.Context, string) (Store, func() error, error)
}

func RunPreview(ctx context.Context, input Input) (declaration.Delta, error) {
	return execute(ctx, input, processlog.DeclarationPreviewed, Preview)
}

func RunApply(ctx context.Context, input Input) (declaration.Delta, error) {
	return execute(ctx, input, processlog.DeclarationApplied, Apply)
}

func execute(
	ctx context.Context,
	input Input,
	recordResult func(processlog.Outcome, processlog.Code) processlog.Record,
	run func(context.Context, Store, declaration.Document) (declaration.Delta, error),
) (declaration.Delta, error) {
	if input.Storage == nil {
		record(input.Recorder, processlog.DependencyFailure(processlog.DependencyUnavailable))
		record(input.Recorder, recordResult(processlog.Failed, processlog.DependencyUnavailable))
		return declaration.Delta{}, errStorageRequired
	}
	document, err := declaration.Read(input.Declaration)
	if err != nil {
		record(input.Recorder, recordResult(processlog.Failed, processlog.DeclarationInvalid))
		return declaration.Delta{}, err
	}
	store, closeStore, err := input.Storage.Open(ctx, input.DatabaseURL)
	if err != nil {
		record(input.Recorder, processlog.StorageFailure(processlog.StorageUnavailable))
		record(input.Recorder, recordResult(processlog.Failed, processlog.StorageUnavailable))
		return declaration.Delta{}, err
	}
	defer closeStorage(closeStore)
	delta, err := run(ctx, store, document)
	if err != nil {
		record(input.Recorder, recordResult(processlog.Failed, processlog.DeclarationInvalid))
		return declaration.Delta{}, err
	}
	record(input.Recorder, recordResult(processlog.Succeeded, ""))
	return delta, nil
}

func closeStorage(closeStore func() error) {
	if closeStore != nil {
		_ = closeStore()
	}
}

func record(recorder processlog.Recorder, item processlog.Record) {
	if recorder != nil {
		recorder.Record(item)
	}
}
