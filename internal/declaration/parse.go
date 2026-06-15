package declaration

import (
	"bytes"
	"encoding/json"
	"strings"

	"gopkg.in/yaml.v3"
)

func Parse(content []byte) (document Document, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if item, ok := recovered.(error); ok {
				document = Document{}
				err = item
				return
			}
			panic(recovered)
		}
	}()
	root, err := parseRoot(content)
	if err != nil {
		return Document{}, err
	}
	document, err = decodeDocument(root)
	if err != nil {
		return Document{}, err
	}
	if err := validate(document); err != nil {
		return Document{}, err
	}
	return document, nil
}

func parseRoot(content []byte) (*yaml.Node, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	var root yaml.Node
	if err := decoder.Decode(&root); err != nil {
		return nil, invalid("parse_error", err.Error())
	}
	if err := decoder.Decode(&yaml.Node{}); err == nil {
		return nil, invalid("multiple_documents", "one document is allowed")
	}
	if len(root.Content) != 1 || root.Content[0].Kind != yaml.MappingNode {
		return nil, invalid("invalid_shape", "top level must be a map")
	}
	if err := rejectUnsafeYAML(root.Content[0]); err != nil {
		return nil, err
	}
	return root.Content[0], nil
}

func rejectUnsafeYAML(node *yaml.Node) error {
	if node.Anchor != "" || node.Kind == yaml.AliasNode {
		return invalid("invalid_yaml_feature", "anchors and aliases are not allowed")
	}
	switch node.Kind {
	case yaml.MappingNode:
		return rejectMapFeatures(node)
	case yaml.SequenceNode:
		return rejectSequenceFeatures(node)
	case yaml.ScalarNode:
		return rejectScalarTag(node)
	default:
		return invalid("invalid_yaml_feature", "node kind is not allowed")
	}
}

func rejectMapFeatures(node *yaml.Node) error {
	seen := map[string]bool{}
	for index := 0; index < len(node.Content); index += 2 {
		key := node.Content[index]
		if key.Value == "<<" {
			return invalid("invalid_yaml_feature", "merge keys are not allowed")
		}
		if seen[key.Value] {
			return invalid("duplicate_key", key.Value)
		}
		seen[key.Value] = true
		if err := rejectUnsafeYAML(key); err != nil {
			return err
		}
		if err := rejectUnsafeYAML(node.Content[index+1]); err != nil {
			return err
		}
	}
	return nil
}

func rejectSequenceFeatures(node *yaml.Node) error {
	for _, item := range node.Content {
		if err := rejectUnsafeYAML(item); err != nil {
			return err
		}
	}
	return nil
}

func rejectScalarTag(node *yaml.Node) error {
	switch node.Tag {
	case "!!str", "!!int", "!!bool", "!!float", "!!null":
		return nil
	default:
		return invalid("invalid_yaml_feature", "custom tags are not allowed")
	}
}

func decodeDocument(root *yaml.Node) (Document, error) {
	fields := mapFields(root)
	if err := requireFields(fields, "version", "schemas", "items", "needs", "nodes", "routes"); err != nil {
		return Document{}, err
	}
	if err := rejectUnknown(fields, "version", "schemas", "items", "needs", "nodes", "routes"); err != nil {
		return Document{}, err
	}
	version, err := intValue(fields["version"], "version")
	if err != nil {
		return Document{}, err
	}
	if version != 1 {
		return Document{}, invalid("invalid_version", "version must be 1")
	}
	return Document{
		Version: version,
		Schemas: must(parseSchemas(fields["schemas"])),
		Items:   must(parseItems(fields["items"])),
		Needs:   must(parseNeeds(fields["needs"])),
		Nodes:   must(parseNodes(fields["nodes"])),
		Routes:  must(parseRoutes(fields["routes"])),
	}, nil
}

func parseSchemas(node *yaml.Node) (map[string]Schema, error) {
	fields, err := keyedMap(node, "schemas")
	if err != nil {
		return nil, err
	}
	schemas := map[string]Schema{}
	for key, value := range fields {
		if err := validateKey("schemas", key); err != nil {
			return nil, err
		}
		item := mapFields(value)
		if err := requireFields(item, "document"); err != nil {
			return nil, err
		}
		if err := rejectUnknown(item, "document"); err != nil {
			return nil, err
		}
		document, err := jsonBytes(item["document"])
		if err != nil {
			return nil, err
		}
		schemas[key] = Schema{Document: document}
	}
	return schemas, nil
}

func parseItems(node *yaml.Node) (map[string]Item, error) {
	fields, err := keyedMap(node, "items")
	if err != nil {
		return nil, err
	}
	items := map[string]Item{}
	for key, value := range fields {
		if err := validateKey("items", key); err != nil {
			return nil, err
		}
		item, err := parseItem(value)
		if err != nil {
			return nil, err
		}
		items[key] = item
	}
	return items, nil
}

func parseItem(node *yaml.Node) (Item, error) {
	fields := mapFields(node)
	if err := requireFields(fields, "schema", "description"); err != nil {
		return Item{}, err
	}
	if err := rejectUnknown(fields, "schema", "description"); err != nil {
		return Item{}, err
	}
	return Item{
		Schema:      mustString(fields["schema"], "schema"),
		Description: mustString(fields["description"], "description"),
	}, nil
}

func parseNeeds(node *yaml.Node) (map[string]Need, error) {
	fields, err := keyedMap(node, "needs")
	if err != nil {
		return nil, err
	}
	needs := map[string]Need{}
	for key, value := range fields {
		if err := validateKey("needs", key); err != nil {
			return nil, err
		}
		need, err := parseNeed(value)
		if err != nil {
			return nil, err
		}
		needs[key] = need
	}
	return needs, nil
}

func parseNeed(node *yaml.Node) (Need, error) {
	fields := mapFields(node)
	if err := requireFields(fields, "description"); err != nil {
		return Need{}, err
	}
	if err := rejectUnknown(fields, "schema", "description"); err != nil {
		return Need{}, err
	}
	need := Need{Description: mustString(fields["description"], "description")}
	if schema, ok := fields["schema"]; ok {
		need.Schema = mustString(schema, "schema")
		need.HasSchema = true
	}
	return need, nil
}

func parseNodes(node *yaml.Node) (map[string]Node, error) {
	fields, err := keyedMap(node, "nodes")
	if err != nil {
		return nil, err
	}
	nodes := map[string]Node{}
	for key, value := range fields {
		if err := validateKey("nodes", key); err != nil {
			return nil, err
		}
		node, err := parseNode(value)
		if err != nil {
			return nil, err
		}
		nodes[key] = node
	}
	return nodes, nil
}

func parseNode(node *yaml.Node) (Node, error) {
	fields := mapFields(node)
	if err := requireFields(fields, "description", "accepts"); err != nil {
		return Node{}, err
	}
	if err := rejectUnknown(fields, "description", "accepts"); err != nil {
		return Node{}, err
	}
	return Node{
		Description: mustString(fields["description"], "description"),
		Accepts:     stringList(fields["accepts"], "accepts"),
	}, nil
}

func parseRoutes(node *yaml.Node) (map[string][]Route, error) {
	fields, err := keyedMap(node, "routes")
	if err != nil {
		return nil, err
	}
	routes := map[string][]Route{}
	for need, value := range fields {
		if err := validateKey("routes", need); err != nil {
			return nil, err
		}
		routes[need] = routeList(value)
	}
	return routes, nil
}

func routeList(node *yaml.Node) []Route {
	if node.Kind != yaml.SequenceNode {
		panic(invalid("invalid_type", "routes entries must be lists"))
	}
	routes := make([]Route, 0, len(node.Content))
	for _, item := range node.Content {
		fields := mapFields(item)
		if err := requireFields(fields, "node"); err != nil {
			panic(err)
		}
		if err := rejectUnknown(fields, "node"); err != nil {
			panic(err)
		}
		routes = append(routes, Route{Node: mustString(fields["node"], "node")})
	}
	return routes
}

func keyedMap(node *yaml.Node, name string) (map[string]*yaml.Node, error) {
	if node.Kind != yaml.MappingNode {
		return nil, invalid("invalid_type", name+" must be a map")
	}
	return mapFields(node), nil
}

func mapFields(node *yaml.Node) map[string]*yaml.Node {
	if node.Kind != yaml.MappingNode {
		panic(invalid("invalid_type", "expected map"))
	}
	fields := map[string]*yaml.Node{}
	for index := 0; index < len(node.Content); index += 2 {
		fields[node.Content[index].Value] = node.Content[index+1]
	}
	return fields
}

func requireFields(fields map[string]*yaml.Node, names ...string) error {
	for _, name := range names {
		if _, ok := fields[name]; !ok {
			return invalid("missing_field", name)
		}
	}
	return nil
}

func rejectUnknown(fields map[string]*yaml.Node, names ...string) error {
	allowed := map[string]bool{}
	for _, name := range names {
		allowed[name] = true
	}
	for name := range fields {
		if !allowed[name] {
			return invalid("unknown_field", name)
		}
	}
	return nil
}

func intValue(node *yaml.Node, field string) (int, error) {
	var value int
	if node.Kind != yaml.ScalarNode || node.Decode(&value) != nil {
		return 0, invalid("invalid_type", field+" must be an integer")
	}
	return value, nil
}

func mustString(node *yaml.Node, field string) string {
	if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		panic(invalid("invalid_type", field+" must be a string"))
	}
	if strings.TrimSpace(node.Value) == "" {
		panic(invalid("empty_value", field))
	}
	return node.Value
}

func stringList(node *yaml.Node, field string) []string {
	if node.Kind != yaml.SequenceNode {
		panic(invalid("invalid_type", field+" must be a list"))
	}
	values := make([]string, 0, len(node.Content))
	for _, item := range node.Content {
		values = append(values, mustString(item, field))
	}
	return values
}

func jsonBytes(node *yaml.Node) ([]byte, error) {
	value, err := jsonValue(node)
	if err != nil {
		return nil, err
	}
	content, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func jsonValue(node *yaml.Node) (any, error) {
	switch node.Kind {
	case yaml.MappingNode:
		return jsonMap(node)
	case yaml.SequenceNode:
		return jsonSequence(node)
	case yaml.ScalarNode:
		return jsonScalar(node)
	default:
		return nil, invalid("invalid_type", "schema document contains unsupported YAML")
	}
}

func jsonMap(node *yaml.Node) (map[string]any, error) {
	result := map[string]any{}
	for index := 0; index < len(node.Content); index += 2 {
		key := node.Content[index]
		if key.Tag != "!!str" {
			return nil, invalid("invalid_type", "schema document keys must be strings")
		}
		value, err := jsonValue(node.Content[index+1])
		if err != nil {
			return nil, err
		}
		result[key.Value] = value
	}
	return result, nil
}

func jsonSequence(node *yaml.Node) ([]any, error) {
	result := make([]any, 0, len(node.Content))
	for _, item := range node.Content {
		value, err := jsonValue(item)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func jsonScalar(node *yaml.Node) (any, error) {
	switch node.Tag {
	case "!!str":
		return node.Value, nil
	case "!!int":
		var value int64
		return value, node.Decode(&value)
	case "!!bool":
		var value bool
		return value, node.Decode(&value)
	case "!!float":
		var value float64
		return value, node.Decode(&value)
	case "!!null":
		return nil, nil
	default:
		return nil, invalid("invalid_type", "schema document scalar is not JSON-compatible")
	}
}

func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}
