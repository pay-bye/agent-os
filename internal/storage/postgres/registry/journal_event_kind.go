package registry

import (
	"database/sql"
	"errors"
	records "github.com/pay-bye/agent-os/internal/registry"
)

func journalEventKindFromRow(
	key records.JournalEventKindKey,
	row rowScanner,
) (records.JournalEventKind, error) {
	var schema sql.NullString
	var description string
	if err := row.Scan(&schema, &description); err != nil {
		return records.JournalEventKind{}, journalEventKindError(key, err)
	}
	return records.NewJournalEventKind(records.JournalEventKindInput{
		Key:         key,
		Schema:      records.SchemaKey(schema.String),
		HasSchema:   schema.Valid,
		Description: description,
	})
}

func journalEventKindError(key records.JournalEventKindKey, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return records.JournalEventKindNotFound(key)
	}
	return err
}
