package registry

import (
	"database/sql"
	"errors"
	records "github.com/pay-bye/agent-os/internal/registry"
)

func needKindFromRow(key records.NeedKindKey, row rowScanner) (records.NeedKind, error) {
	var schemaKey sql.NullString
	var description string
	if err := row.Scan(&schemaKey, &description); err != nil {
		return records.NeedKind{}, needKindError(key, err)
	}
	if schemaKey.Valid {
		return records.NewNeedKindWithSchema(key, records.SchemaKey(schemaKey.String), description), nil
	}
	return records.NewNeedKind(key, description), nil
}

func needKindError(key records.NeedKindKey, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return records.NeedKindNotFound(key)
	}
	return err
}
