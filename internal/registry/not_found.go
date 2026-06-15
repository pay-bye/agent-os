package registry

import (
	"errors"
	"fmt"
)

type notFoundError struct {
	family string
	key    string
}

func (e notFoundError) Error() string {
	return fmt.Sprintf("registry %s not found: key=%q", e.family, e.key)
}

func SchemaDocumentNotFound(key SchemaKey) error {
	return missing("schema document", key.String())
}

func ItemKindNotFound(key ItemKindKey) error {
	return missing("item kind", key.String())
}

func NeedKindNotFound(key NeedKindKey) error {
	return missing("need kind", key.String())
}

func NodeNotFound(key NodeKey) error {
	return missing("node", key.String())
}

func ChannelNotFound(key ChannelKey) error {
	return missing("channel", key.String())
}

func JournalEventKindNotFound(key JournalEventKindKey) error {
	return missing("journal event kind", key.String())
}

func IsNotFound(err error) bool {
	var target notFoundError
	return errors.As(err, &target)
}

func missing(family string, key string) notFoundError {
	return notFoundError{family: family, key: key}
}
