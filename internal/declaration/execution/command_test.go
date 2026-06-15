package execution

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/internal/declaration"
	"github.com/pay-bye/agent-os/internal/processlog"
	"github.com/pay-bye/agent-os/internal/storage/postgres"
)

func TestRunPreviewRequiresStorage(t *testing.T) {
	_, err := RunPreview(context.Background(), Input{})

	requireError(t, err, "storage_required")
}

func TestRunPreviewReadsDeclarationBeforeOpeningStorage(t *testing.T) {
	storage := &recordingStorage{store: &recordingStore{}}
	path := filepath.Join(t.TempDir(), "missing.yaml")

	_, err := RunPreview(context.Background(), Input{
		Declaration: path,
		Storage:     storage,
	})

	if err == nil {
		t.Fatal("preview succeeded with missing declaration")
	}
	if storage.opened {
		t.Fatal("storage opened before declaration read")
	}
}

func TestRunApplyUsesStorage(t *testing.T) {
	store := &recordingStore{}
	storage := &recordingStorage{store: store}
	path := writeDeclaration(t)

	_, err := RunApply(context.Background(), Input{
		DatabaseURL: "postgres://u:p@host/db",
		Declaration: path,
		Storage:     storage,
	})

	if err != nil {
		t.Fatal(err)
	}
	if storage.databaseURL != "postgres://u:p@host/db" {
		t.Fatalf("database URL = %q", storage.databaseURL)
	}
	if !storage.closed {
		t.Fatal("expected storage close")
	}
	if !store.committed {
		t.Fatal("expected commit")
	}
}

func TestRunPreviewRecordsDeclarationSuccess(t *testing.T) {
	recorder := &recordingRecorder{}
	path := writeDeclaration(t)

	_, err := RunPreview(context.Background(), Input{
		DatabaseURL: "postgres://u:p@host/db",
		Declaration: path,
		Storage:     &recordingStorage{store: &recordingStore{}},
		Recorder:    recorder,
	})

	if err != nil {
		t.Fatal(err)
	}
	requireRecordedOperations(t, recorder.records, processlog.DeclarationPreview)
	if recorder.records[0].Outcome != processlog.Succeeded {
		t.Fatalf("outcome = %q, want succeeded", recorder.records[0].Outcome)
	}
}

func TestRunPreviewRecordsStorageFailureWithoutSensitiveMaterial(t *testing.T) {
	recorder := &recordingRecorder{}
	sentinel := "postgres://user:secret@host/db"

	_, err := RunPreview(context.Background(), Input{
		DatabaseURL: sentinel,
		Declaration: writeDeclaration(t),
		Storage:     failingStorage{err: errors.New("connect failed: " + sentinel)},
		Recorder:    recorder,
	})

	if err == nil {
		t.Fatal("expected storage failure")
	}
	requireRecordedOperations(t, recorder.records, processlog.StorageError, processlog.DeclarationPreview)
	if recorder.records[0].ErrorCode != processlog.StorageUnavailable {
		t.Fatalf("storage error code = %q, want %q", recorder.records[0].ErrorCode, processlog.StorageUnavailable)
	}
	if recorder.records[1].ErrorCode != processlog.StorageUnavailable {
		t.Fatalf("declaration error code = %q, want %q", recorder.records[1].ErrorCode, processlog.StorageUnavailable)
	}
	requireRecordsExclude(t, recorder.records, "secret", sentinel)
}

type recordingStorage struct {
	store       *recordingStore
	databaseURL string
	opened      bool
	closed      bool
}

func (s *recordingStorage) Open(_ context.Context, databaseURL string) (Store, func() error, error) {
	s.databaseURL = databaseURL
	s.opened = true
	return s.store, func() error {
		s.closed = true
		return nil
	}, nil
}

type failingStorage struct {
	err error
}

func (s failingStorage) Open(context.Context, string) (Store, func() error, error) {
	return nil, nil, s.err
}

type recordingRecorder struct {
	records []processlog.Record
}

func (r *recordingRecorder) Record(record processlog.Record) {
	r.records = append(r.records, record)
}

func requireRecordedOperations(t *testing.T, records []processlog.Record, operations ...processlog.Operation) {
	t.Helper()

	if len(records) != len(operations) {
		t.Fatalf("records = %+v, want operations %v", records, operations)
	}
	for index, operation := range operations {
		if records[index].Operation != operation {
			t.Fatalf("record %d operation = %q, want %q", index, records[index].Operation, operation)
		}
	}
}

func requireRecordsExclude(t *testing.T, records []processlog.Record, forbidden ...string) {
	t.Helper()

	encoded, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range forbidden {
		if strings.Contains(string(encoded), value) {
			t.Fatalf("records exposed %q: %s", value, string(encoded))
		}
	}
}

type recordingStore struct {
	committed bool
}

func (s *recordingStore) Snapshot(context.Context) (postgres.Catalog, error) {
	return postgres.Catalog{}, nil
}

func (s *recordingStore) Install(context.Context, postgres.Catalog) error {
	return nil
}

func (s *recordingStore) Reconcile(context.Context, declaration.Delta) error {
	return nil
}

func (s *recordingStore) Commit() error {
	s.committed = true
	return nil
}

func (s *recordingStore) Rollback() error {
	return nil
}

func writeDeclaration(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "vocabulary.yaml")
	if err := os.WriteFile(path, []byte(validDeclaration()), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func validDeclaration() string {
	return `version: 1
schemas: {x01: {document: {type: object}}}
items: {x08: {schema: x01, description: x21}}
needs: {x12: {schema: x01, description: x22}}
nodes: {x17: {description: x23, accepts: [x12]}}
routes: {x12: [{node: x17}]}
`
}
