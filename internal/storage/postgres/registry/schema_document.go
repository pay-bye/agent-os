package registry

import (
	"database/sql"
	"errors"
	records "github.com/pay-bye/agent-os/internal/registry"
)

func schemaDocumentFromRow(key records.SchemaKey, row rowScanner) (records.SchemaDocument, error) {
	var document []byte
	if err := row.Scan(&document); err != nil {
		return records.SchemaDocument{}, schemaDocumentError(key, err)
	}
	return records.NewSchemaDocument(key, document), nil
}

func schemaDocumentError(key records.SchemaKey, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return records.SchemaDocumentNotFound(key)
	}
	return err
}
