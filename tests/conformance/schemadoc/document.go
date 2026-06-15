package schemadoc

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

type Document struct {
	Type                 string              `json:"type"`
	Required             []string            `json:"required"`
	Properties           map[string]Document `json:"properties"`
	Items                *Document           `json:"items"`
	OneOf                []Document          `json:"oneOf"`
	MinItems             int                 `json:"minItems"`
	MaxItems             int                 `json:"maxItems"`
	UniqueItems          bool                `json:"uniqueItems"`
	MinLength            int                 `json:"minLength"`
	Minimum              int                 `json:"minimum"`
	Pattern              string              `json:"pattern"`
	Const                *json.RawMessage    `json:"const"`
	AdditionalProperties *bool               `json:"additionalProperties"`
}

func Read(t testing.TB, rootPath string, relativePath string) Document {
	t.Helper()

	content, err := read(rootPath, relativePath)
	if err != nil {
		t.Fatal(err)
	}
	var document Document
	if err := json.Unmarshal(content, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func Validate(document Document, payload map[string]any) error {
	return validateNode("payload", payload, document)
}

func read(rootPath string, relativePath string) ([]byte, error) {
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	file, err := root.Open(relativePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func validateNode(name string, value any, document Document) error {
	if len(document.OneOf) > 0 {
		return validateOneOf(name, value, document.OneOf)
	}
	if !constMatches(document.Const, value) {
		return fmt.Errorf("%s must equal %s", name, string(*document.Const))
	}
	if document.Type != "object" {
		return validateScalar(name, value, document)
	}
	payload, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("%s must be an object", name)
	}
	for _, field := range document.Required {
		if _, ok := payload[field]; !ok {
			return fmt.Errorf("required field %s is missing", field)
		}
	}
	for field, value := range payload {
		fieldDocument, ok := document.Properties[field]
		if !ok {
			if rejectsUnknownFields(document) {
				return fmt.Errorf("%s is not allowed", field)
			}
			continue
		}
		if err := validateNode(field, value, fieldDocument); err != nil {
			return err
		}
	}
	return nil
}

func validateScalar(field string, value any, document Document) error {
	switch document.Type {
	case "string":
		return requireString(field, value, document)
	case "integer":
		return requireNumber(field, value, document.Minimum)
	case "number":
		return requireRealNumber(field, value, document.Minimum)
	case "object":
		return requireObject(field, value)
	case "array":
		return requireArray(field, value, document)
	case "boolean":
		return requireBoolean(field, value)
	default:
		return fmt.Errorf("%s has unsupported type %q", field, document.Type)
	}
}

func validateOneOf(name string, value any, variants []Document) error {
	matches := 0
	messages := make([]string, 0, len(variants))
	for _, variant := range variants {
		if err := validateNode(name, value, variant); err != nil {
			messages = append(messages, err.Error())
			continue
		}
		matches++
	}
	if matches == 1 {
		return nil
	}
	return fmt.Errorf("%s must match exactly one variant: %s", name, strings.Join(messages, "; "))
}

func constMatches(raw *json.RawMessage, value any) bool {
	if raw == nil {
		return true
	}
	var want any
	if err := json.Unmarshal(*raw, &want); err != nil {
		return false
	}
	return reflect.DeepEqual(want, value)
}

func rejectsUnknownFields(document Document) bool {
	return document.AdditionalProperties != nil && !*document.AdditionalProperties
}

func requireString(field string, value any, document Document) error {
	text, ok := value.(string)
	if !ok {
		return fmt.Errorf("%s must be a string", field)
	}
	if len(text) < document.MinLength {
		return fmt.Errorf("%s must be a non-empty string", field)
	}
	if document.Pattern != "" && !regexp.MustCompile(document.Pattern).MatchString(text) {
		return fmt.Errorf("%s must match %s", field, document.Pattern)
	}
	return nil
}

func requireNumber(field string, value any, minimum int) error {
	number, ok := value.(float64)
	if !ok || number != float64(int64(number)) {
		return fmt.Errorf("%s must be an integer", field)
	}
	if int(number) < minimum {
		return fmt.Errorf("%s must be at least %d", field, minimum)
	}
	return nil
}

func requireRealNumber(field string, value any, minimum int) error {
	number, ok := value.(float64)
	if !ok {
		return fmt.Errorf("%s must be a number", field)
	}
	if number < float64(minimum) {
		return fmt.Errorf("%s must be at least %d", field, minimum)
	}
	return nil
}

func requireObject(field string, value any) error {
	if _, ok := value.(map[string]any); !ok {
		return fmt.Errorf("%s must be an object", field)
	}
	return nil
}

func requireArray(field string, value any, document Document) error {
	values, ok := value.([]any)
	if !ok {
		return fmt.Errorf("%s must be an array", field)
	}
	if len(values) < document.MinItems {
		return fmt.Errorf("%s must contain at least %d items", field, document.MinItems)
	}
	if document.MaxItems > 0 && len(values) > document.MaxItems {
		return fmt.Errorf("%s must contain at most %d items", field, document.MaxItems)
	}
	if document.UniqueItems {
		if index := firstDuplicateIndex(values); index >= 0 {
			return fmt.Errorf("%s[%d] must be unique", field, index)
		}
	}
	if len(values) > 0 && document.Items == nil {
		return fmt.Errorf("%s item schema is missing", field)
	}
	for index, item := range values {
		if err := validateNode(fmt.Sprintf("%s[%d]", field, index), item, *document.Items); err != nil {
			return err
		}
	}
	return nil
}

func firstDuplicateIndex(values []any) int {
	for index, value := range values {
		if containsValue(values[:index], value) {
			return index
		}
	}
	return -1
}

func containsValue(values []any, value any) bool {
	for _, existing := range values {
		if reflect.DeepEqual(existing, value) {
			return true
		}
	}
	return false
}

func requireBoolean(field string, value any) error {
	if _, ok := value.(bool); !ok {
		return fmt.Errorf("%s must be a boolean", field)
	}
	return nil
}
