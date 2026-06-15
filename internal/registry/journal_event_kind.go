package registry

import (
	"errors"
)

var (
	ErrEmptyEventKindKey     = errors.New("journal event kind key is empty")
	ErrEmptySchemaKey        = errors.New("schema key is empty")
	ErrEmptyEventDescription = errors.New("journal event kind description is empty")
)

type JournalEventKindKey string

func (k JournalEventKindKey) String() string {
	return string(k)
}

type JournalEventKindInput struct {
	Key         JournalEventKindKey
	Schema      SchemaKey
	HasSchema   bool
	Description string
}

type JournalEventKind struct {
	key         JournalEventKindKey
	schema      SchemaKey
	hasSchema   bool
	description string
}

func NewJournalEventKind(input JournalEventKindInput) (JournalEventKind, error) {
	if err := validateJournalEventKindInput(input); err != nil {
		return JournalEventKind{}, err
	}
	return JournalEventKind{
		key:         input.Key,
		schema:      input.Schema,
		hasSchema:   input.HasSchema,
		description: input.Description,
	}, nil
}

func (k JournalEventKind) Key() JournalEventKindKey {
	return k.key
}

func (k JournalEventKind) Description() string {
	return k.description
}

func (k JournalEventKind) Schema() (SchemaKey, bool) {
	return k.schema, k.hasSchema
}

func validateJournalEventKindInput(input JournalEventKindInput) error {
	if blank(input.Key.String()) {
		return ErrEmptyEventKindKey
	}
	if input.HasSchema && blank(input.Schema.String()) {
		return ErrEmptySchemaKey
	}
	if blank(input.Description) {
		return ErrEmptyEventDescription
	}
	return nil
}
