package registry

type SchemaKey string

func (k SchemaKey) String() string {
	return string(k)
}

type SchemaDocument struct {
	key      SchemaKey
	document []byte
}

func NewSchemaDocument(key SchemaKey, document []byte) SchemaDocument {
	return SchemaDocument{key: key, document: copyBytes(document)}
}

func (d SchemaDocument) Key() SchemaKey {
	return d.key
}

func (d SchemaDocument) Document() []byte {
	return copyBytes(d.document)
}

func copyBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	return append([]byte(nil), value...)
}
