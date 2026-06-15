package registry

type ItemKindKey string

func (k ItemKindKey) String() string {
	return string(k)
}

type ItemKind struct {
	key         ItemKindKey
	schemaKey   SchemaKey
	description string
}

func NewItemKind(key ItemKindKey, schemaKey SchemaKey, description string) ItemKind {
	return ItemKind{key: key, schemaKey: schemaKey, description: description}
}

func (k ItemKind) Key() ItemKindKey {
	return k.key
}

func (k ItemKind) SchemaKey() SchemaKey {
	return k.schemaKey
}

func (k ItemKind) Description() string {
	return k.description
}
