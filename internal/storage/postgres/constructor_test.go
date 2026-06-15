package postgres

import (
	"database/sql"
	"testing"
)

func TestConstructorsInstallDatabaseOperations(t *testing.T) {
	var db *sql.DB

	stores := []any{
		NewChannel(db),
		NewJournal(db),
		NewRegistry(db),
		NewKernel(db, WithSearchPath("x01")),
		NewMetrics(db),
	}

	for index, store := range stores {
		if store == nil {
			t.Fatalf("store[%d] is nil", index)
		}
	}
}
