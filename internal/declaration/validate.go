package declaration

func validate(document Document) error {
	for key, item := range document.Items {
		if _, ok := document.Schemas[item.Schema]; !ok {
			return invalid("unknown_reference", key+".schema")
		}
	}
	for key, need := range document.Needs {
		if need.HasSchema {
			if _, ok := document.Schemas[need.Schema]; !ok {
				return invalid("unknown_reference", key+".schema")
			}
		}
	}
	for key, node := range document.Nodes {
		if err := validateAccepts(document.Needs, key, node.Accepts); err != nil {
			return err
		}
	}
	return validateRoutes(document)
}

func validateAccepts(needs map[string]Need, node string, accepts []string) error {
	seen := map[string]bool{}
	for _, need := range accepts {
		if err := validateKey("accepts", need); err != nil {
			return err
		}
		if seen[need] {
			return invalid("duplicate_key", node+".accepts")
		}
		seen[need] = true
		if _, ok := needs[need]; !ok {
			return invalid("unknown_reference", node+".accepts")
		}
	}
	return nil
}

func validateRoutes(document Document) error {
	for need, routes := range document.Routes {
		if _, ok := document.Needs[need]; !ok {
			return invalid("unknown_reference", need)
		}
		for _, route := range routes {
			node, ok := document.Nodes[route.Node]
			if !ok {
				return invalid("unknown_reference", route.Node)
			}
			if !contains(node.Accepts, need) {
				return invalid("route_capability", need+" -> "+route.Node)
			}
		}
	}
	return nil
}

func validateKey(_ string, value string) error {
	if !validKey(value) {
		return invalid("invalid_key", value)
	}
	return nil
}

func validKey(value string) bool {
	if value == "" {
		return false
	}
	for index, char := range value {
		if index == 0 && (char == '_' || char == '-' || char == '.') {
			return false
		}
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '_' || char == '-' || char == '.' {
			continue
		}
		return false
	}
	return true
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
