package registry

import (
	"database/sql"
	"errors"
	records "github.com/pay-bye/agent-os/internal/registry"
)

func itemKindFromRow(key records.ItemKindKey, row rowScanner) (records.ItemKind, error) {
	var schemaKey string
	var description string
	if err := row.Scan(&schemaKey, &description); err != nil {
		return records.ItemKind{}, itemKindError(key, err)
	}
	return records.NewItemKind(key, records.SchemaKey(schemaKey), description), nil
}

func itemKindError(key records.ItemKindKey, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return records.ItemKindNotFound(key)
	}
	return err
}
