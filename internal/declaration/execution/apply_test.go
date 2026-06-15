package execution

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/internal/declaration"
	"github.com/pay-bye/agent-os/internal/storage/postgres"
)

func TestApplyUsesSingleTransaction(t *testing.T) {
	store := &orderedStore{installErr: errors.New("install failed")}
	_, err := Apply(context.Background(), store, mustParse(t, validDocument()))

	requireError(t, err, "install failed")
	requireOrder(t, store.calls, "snapshot", "reconcile", "install", "rollback")
}

func TestApplyRollsBackSnapshotFailure(t *testing.T) {
	store := &orderedStore{snapshotErr: errors.New("snapshot failed")}

	_, err := Apply(context.Background(), store, mustParse(t, validDocument()))

	requireError(t, err, "snapshot failed")
	requireOrder(t, store.calls, "snapshot", "rollback")
}

func TestApplyRollsBackReconcileFailure(t *testing.T) {
	store := &orderedStore{reconcileErr: errors.New("reconcile failed")}

	_, err := Apply(context.Background(), store, mustParse(t, validDocument()))

	requireError(t, err, "reconcile failed")
	requireOrder(t, store.calls, "snapshot", "reconcile", "rollback")
}

func TestApplyRollsBackCommitFailure(t *testing.T) {
	store := &orderedStore{commitErr: errors.New("commit failed")}

	_, err := Apply(context.Background(), store, mustParse(t, validDocument()))

	requireError(t, err, "commit failed")
	requireOrder(t, store.calls, "snapshot", "reconcile", "install", "commit", "rollback")
}

func TestApplyReconcilesBeforeInstallAndCommit(t *testing.T) {
	store := &orderedStore{}

	delta, err := Apply(context.Background(), store, mustParse(t, validDocument()))
	if err != nil {
		t.Fatal(err)
	}

	requireOrder(t, store.calls, "snapshot", "reconcile", "install", "commit")
	if len(delta.Additions) == 0 {
		t.Fatalf("delta = %+v, want additions", delta)
	}
	if store.reconciled == nil {
		t.Fatal("expected reconciliation delta")
	}
	if store.installed.Schemas == nil {
		t.Fatal("expected installed catalog")
	}
}

func TestApplyRollsBackUnsafeDelta(t *testing.T) {
	document := mustParse(t, validDocument())
	store := &orderedStore{catalog: conflictingCatalog(document)}

	delta, err := Apply(context.Background(), store, document)

	requireError(t, err, "unsafe declaration delta")
	requireOrder(t, store.calls, "snapshot", "rollback")
	if delta.Installable {
		t.Fatalf("delta = %+v, want unsafe", delta)
	}
}

func TestPreviewRollsBackAfterDelta(t *testing.T) {
	store := &orderedStore{}

	delta, err := Preview(context.Background(), store, mustParse(t, validDocument()))
	if err != nil {
		t.Fatal(err)
	}

	requireOrder(t, store.calls, "snapshot", "rollback")
	if len(delta.Additions) == 0 {
		t.Fatalf("delta = %+v, want additions", delta)
	}
}

func TestPreviewRollsBackSnapshotFailure(t *testing.T) {
	store := &orderedStore{snapshotErr: errors.New("snapshot failed")}

	_, err := Preview(context.Background(), store, mustParse(t, validDocument()))

	requireError(t, err, "snapshot failed")
	requireOrder(t, store.calls, "snapshot", "rollback")
}

type orderedStore struct {
	calls        []string
	catalog      postgres.Catalog
	reconciled   *declaration.Delta
	installed    postgres.Catalog
	snapshotErr  error
	reconcileErr error
	installErr   error
	commitErr    error
}

func (s *orderedStore) Snapshot(context.Context) (postgres.Catalog, error) {
	s.calls = append(s.calls, "snapshot")
	if s.snapshotErr != nil {
		return postgres.Catalog{}, s.snapshotErr
	}
	if s.catalog.Schemas != nil {
		return s.catalog, nil
	}
	return postgres.Catalog{}, nil
}

func (s *orderedStore) Reconcile(_ context.Context, delta declaration.Delta) error {
	s.calls = append(s.calls, "reconcile")
	s.reconciled = &delta
	return s.reconcileErr
}

func (s *orderedStore) Install(_ context.Context, catalog postgres.Catalog) error {
	s.calls = append(s.calls, "install")
	s.installed = catalog
	return s.installErr
}

func (s *orderedStore) Commit() error {
	s.calls = append(s.calls, "commit")
	return s.commitErr
}

func (s *orderedStore) Rollback() error {
	s.calls = append(s.calls, "rollback")
	return nil
}

func requireOrder(t *testing.T, got []string, want ...string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("calls = %v, want %v", got, want)
	}
	for index := range got {
		if got[index] != want[index] {
			t.Fatalf("call[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

func validDocument() string {
	return `version: 1
schemas:
  x01:
    document:
      type: object
items:
  x08:
    schema: x01
    description: x21
needs:
  x12:
    schema: x01
    description: x22
nodes:
  x17:
    description: x23
    accepts:
      - x12
routes:
  x12:
    - node: x17
`
}

func mustParse(t *testing.T, body string) declaration.Document {
	t.Helper()

	document, err := declaration.Parse([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func conflictingCatalog(document declaration.Document) postgres.Catalog {
	catalog := document.Vocabulary()
	catalog.Items[0].Description = "different"
	return catalog
}

func requireError(t *testing.T, err error, text string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error containing %q", text)
	}
	if !strings.Contains(err.Error(), text) {
		t.Fatalf("error = %v, want %q", err, text)
	}
}
