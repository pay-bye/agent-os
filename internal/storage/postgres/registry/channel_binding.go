package registry

import (
	"database/sql"
	"errors"
	records "github.com/pay-bye/agent-os/internal/registry"
)

func channelFromRow(key records.ChannelKey, row rowScanner) (records.Channel, error) {
	var node string
	var description string
	if err := row.Scan(&node, &description); err != nil {
		return records.Channel{}, channelError(key, err)
	}
	return records.NewChannel(records.ChannelInput{
		Key:         key,
		Node:        records.NodeKey(node),
		Description: description,
	})
}

func channelError(key records.ChannelKey, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return records.ChannelNotFound(key)
	}
	return err
}
