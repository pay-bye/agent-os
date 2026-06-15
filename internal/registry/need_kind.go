package registry

type NeedKindKey string

func (k NeedKindKey) String() string {
	return string(k)
}

type NeedKind struct {
	key         NeedKindKey
	schemaKey   SchemaKey
	hasSchema   bool
	description string
}

func NewNeedKind(key NeedKindKey, description string) NeedKind {
	return NeedKind{key: key, description: description}
}

func (k NeedKind) Key() NeedKindKey {
	return k.key
}

func (k NeedKind) Description() string {
	return k.description
}

func (k NeedKind) SchemaKey() (SchemaKey, bool) {
	return k.schemaKey, k.hasSchema
}

func NewNeedKindWithSchema(key NeedKindKey, schemaKey SchemaKey, description string) NeedKind {
	return NeedKind{key: key, schemaKey: schemaKey, hasSchema: true, description: description}
}
