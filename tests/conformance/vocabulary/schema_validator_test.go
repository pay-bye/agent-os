package vocabulary_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

type schemaDocument struct {
	ID                   string                    `json:"$id,omitempty"`
	Ref                  string                    `json:"$ref,omitempty"`
	Defs                 map[string]schemaDocument `json:"$defs,omitempty"`
	Type                 string                    `json:"type,omitempty"`
	Required             []string                  `json:"required,omitempty"`
	Properties           map[string]schemaDocument `json:"properties,omitempty"`
	PropertyNames        *schemaDocument           `json:"propertyNames,omitempty"`
	Items                *schemaDocument           `json:"items,omitempty"`
	MinLength            int                       `json:"minLength,omitempty"`
	Pattern              string                    `json:"pattern,omitempty"`
	Const                *json.RawMessage          `json:"const,omitempty"`
	UniqueItems          bool                      `json:"uniqueItems,omitempty"`
	AdditionalProperties *schemaDocument           `json:"additionalProperties,omitempty"`
	RejectUnknown        bool
}

func (s *schemaDocument) UnmarshalJSON(content []byte) error {
	var boolean bool
	if err := json.Unmarshal(content, &boolean); err == nil {
		s.RejectUnknown = !boolean
		return nil
	}
	type document schemaDocument
	return json.Unmarshal(content, (*document)(s))
}

func readSchema(t *testing.T) schemaDocument {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(findRoot(t), "contracts", "vocabulary", "v1.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema schemaDocument
	if err := json.Unmarshal(content, &schema); err != nil {
		t.Fatal(err)
	}
	return schema
}

func validate(schema schemaDocument, value any) error {
	return validator{defs: schema.Defs}.validate("vocabulary", value, schema)
}

type validator struct {
	defs map[string]schemaDocument
}

func (v validator) validate(name string, value any, schema schemaDocument) error {
	if schema.Ref != "" {
		return v.validate(name, value, v.schema(schema.Ref))
	}
	if schema.Const != nil && !matchesConst(*schema.Const, value) {
		return fmt.Errorf("%s must match const", name)
	}
	switch schema.Type {
	case "":
		return nil
	case "object":
		return v.validateObject(name, value, schema)
	case "array":
		return v.validateArray(name, value, schema)
	case "string":
		return validateString(name, value, schema)
	case "integer":
		return validateInteger(name, value)
	default:
		return fmt.Errorf("%s has unsupported type %q", name, schema.Type)
	}
}

func (v validator) schema(ref string) schemaDocument {
	name := strings.TrimPrefix(ref, "#/$defs/")
	return v.defs[name]
}

func (v validator) validateObject(name string, value any, schema schemaDocument) error {
	fields, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("%s must be an object", name)
	}
	for _, field := range schema.Required {
		if _, ok := fields[field]; !ok {
			return fmt.Errorf("%s.%s is required", name, field)
		}
	}
	for key, item := range fields {
		if err := v.validatePropertyName(name, key, schema); err != nil {
			return err
		}
		if err := v.validateField(name, key, item, schema); err != nil {
			return err
		}
	}
	return nil
}

func (v validator) validatePropertyName(name string, key string, schema schemaDocument) error {
	if schema.PropertyNames == nil {
		return nil
	}
	return v.validate(name+" key", key, *schema.PropertyNames)
}

func (v validator) validateField(name string, key string, value any, schema schemaDocument) error {
	if fieldSchema, ok := schema.Properties[key]; ok {
		return v.validate(name+"."+key, value, fieldSchema)
	}
	if schema.AdditionalProperties == nil || schema.AdditionalProperties.RejectUnknown {
		return fmt.Errorf("%s.%s is not allowed", name, key)
	}
	return v.validate(name+"."+key, value, *schema.AdditionalProperties)
}

func (v validator) validateArray(name string, value any, schema schemaDocument) error {
	items, ok := value.([]any)
	if !ok {
		return fmt.Errorf("%s must be an array", name)
	}
	if schema.UniqueItems && hasDuplicate(items) {
		return fmt.Errorf("%s must contain unique items", name)
	}
	for index, item := range items {
		if schema.Items == nil {
			return fmt.Errorf("%s items schema is missing", name)
		}
		if err := v.validate(fmt.Sprintf("%s[%d]", name, index), item, *schema.Items); err != nil {
			return err
		}
	}
	return nil
}

func validateString(name string, value any, schema schemaDocument) error {
	text, ok := value.(string)
	if !ok {
		return fmt.Errorf("%s must be a string", name)
	}
	if len(text) < schema.MinLength {
		return fmt.Errorf("%s must be non-empty", name)
	}
	if schema.Pattern != "" && !regexp.MustCompile(schema.Pattern).MatchString(text) {
		return fmt.Errorf("%s must match %s", name, schema.Pattern)
	}
	return nil
}

func validateInteger(name string, value any) error {
	if _, ok := value.(int); ok {
		return nil
	}
	return fmt.Errorf("%s must be an integer", name)
}

func matchesConst(raw json.RawMessage, value any) bool {
	var want any
	if err := json.Unmarshal(raw, &want); err != nil {
		return false
	}
	if number, ok := want.(float64); ok {
		return value == int(number)
	}
	return reflect.DeepEqual(want, value)
}

func hasDuplicate(items []any) bool {
	seen := map[any]bool{}
	for _, item := range items {
		if seen[item] {
			return true
		}
		seen[item] = true
	}
	return false
}
