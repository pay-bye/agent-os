package kernel

import (
	"database/sql"

	channelstore "github.com/pay-bye/agent-os/internal/storage/postgres/channel"
	eventstore "github.com/pay-bye/agent-os/internal/storage/postgres/journal"
	registrystore "github.com/pay-bye/agent-os/internal/storage/postgres/registry"
)

type journalStore = eventstore.Store

func newChannel(tx *sql.Tx) *channelstore.Store {
	return channelstore.New(tx)
}

func newJournal(tx *sql.Tx) *journalStore {
	return eventstore.New(tx)
}

func newRegistry(tx *sql.Tx) *registrystore.Store {
	return registrystore.New(tx)
}
