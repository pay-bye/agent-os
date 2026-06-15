package execution

import (
	"context"

	"github.com/pay-bye/agent-os/internal/declaration"
)

func Apply(ctx context.Context, store Store, document declaration.Document) (declaration.Delta, error) {
	current, err := store.Snapshot(ctx)
	if err != nil {
		_ = store.Rollback()
		return declaration.Delta{}, err
	}
	delta := declaration.BuildDelta(current, document.Vocabulary())
	if !delta.Installable {
		_ = store.Rollback()
		return delta, declaration.ErrUnsafeDelta
	}
	if err := store.Reconcile(ctx, delta); err != nil {
		_ = store.Rollback()
		return declaration.Delta{}, err
	}
	if err := store.Install(ctx, document.Vocabulary()); err != nil {
		_ = store.Rollback()
		return declaration.Delta{}, err
	}
	if err := store.Commit(); err != nil {
		_ = store.Rollback()
		return declaration.Delta{}, err
	}
	return delta, nil
}
