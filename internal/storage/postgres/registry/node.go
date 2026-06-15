package registry

import (
	"database/sql"
	"errors"
	records "github.com/pay-bye/agent-os/internal/registry"
)

func nodeRowValues(key records.NodeKey, row rowScanner) (string, records.ChannelKey, error) {
	var description string
	var channel string
	if err := row.Scan(&description, &channel); err != nil {
		return "", "", nodeError(key, err)
	}
	return description, records.ChannelKey(channel), nil
}

func nodeError(key records.NodeKey, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return records.NodeNotFound(key)
	}
	return err
}
