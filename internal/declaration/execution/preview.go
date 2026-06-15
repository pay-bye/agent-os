package execution

import (
	"context"

	"github.com/pay-bye/agent-os/internal/declaration"
	"github.com/pay-bye/agent-os/internal/storage/postgres"
)

type Store interface {
	Snapshot(context.Context) (postgres.Catalog, error)
	Install(context.Context, postgres.Catalog) error
	Reconcile(context.Context, declaration.Delta) error
	Commit() error
	Rollback() error
}

func Preview(ctx context.Context, store Store, document declaration.Document) (declaration.Delta, error) {
	current, err := store.Snapshot(ctx)
	if err != nil {
		_ = store.Rollback()
		return declaration.Delta{}, err
	}
	delta := declaration.BuildDelta(current, document.Vocabulary())
	return delta, store.Rollback()
}
