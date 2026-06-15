package checks

import (
	"go/ast"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestReadmodelPostgresFactsDoNotOwnJSONShape(t *testing.T) {
	path := filepath.Join(findRoot(t), "internal", "readmodel", "postgres", "operations.go")
	file, err := parseGoFile(path)
	if err != nil {
		t.Fatal(err)
	}

	violations := jsonTagViolations(file)
	if len(violations) > 0 {
		t.Fatalf("readmodel postgres facts own response JSON tags: %s", strings.Join(violations, ", "))
	}
}

func jsonTagViolations(file *ast.File) []string {
	var violations []string
	ast.Inspect(file, func(node ast.Node) bool {
		typeSpec, ok := node.(*ast.TypeSpec)
		if !ok {
			return true
		}
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return false
		}
		violations = append(violations, taggedFields(typeSpec.Name.Name, structType)...)
		return false
	})
	return violations
}

func taggedFields(typeName string, structType *ast.StructType) []string {
	var violations []string
	for _, field := range structType.Fields.List {
		if !hasJSONTag(field) {
			continue
		}
		for _, name := range fieldNames(field) {
			violations = append(violations, typeName+"."+name)
		}
	}
	return violations
}

func hasJSONTag(field *ast.Field) bool {
	if field.Tag == nil {
		return false
	}
	value, err := strconv.Unquote(field.Tag.Value)
	if err != nil {
		return false
	}
	_, ok := reflect.StructTag(value).Lookup("json")
	return ok
}
