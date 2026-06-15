package release_test

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

type payloadCase struct {
	name string
	body map[string]any
}

type schemaDocument struct {
	ID                   string                    `json:"$id,omitempty"`
	Ref                  string                    `json:"$ref,omitempty"`
	Defs                 map[string]schemaDocument `json:"$defs,omitempty"`
	Type                 string                    `json:"type,omitempty"`
	Required             []string                  `json:"required,omitempty"`
	Properties           map[string]schemaDocument `json:"properties,omitempty"`
	Items                *schemaDocument           `json:"items,omitempty"`
	MinItems             int                       `json:"minItems,omitempty"`
	MinLength            int                       `json:"minLength,omitempty"`
	Pattern              string                    `json:"pattern,omitempty"`
	Const                *json.RawMessage          `json:"const,omitempty"`
	Enum                 []json.RawMessage         `json:"enum,omitempty"`
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

func invalidPayloadCases() []payloadCase {
	return []payloadCase{
		{name: "missing signatures", body: without(validPayload(), "signatures")},
		{name: "invalid artifact enum value", body: withArtifactField("os", "opaque")},
		{name: "invalid image enum value", body: withImageField("platform", "opaque/platform")},
		{name: "unknown reference property", body: withReferenceField("signatures", "extra", "value")},
	}
}

func validPayload() map[string]any {
	return map[string]any{
		"version": int(1),
		"tag":     "v0.1.0-rc.1",
		"commit":  "0123456789abcdef",
		"source": map[string]any{
			"repository": "example/agent-os",
			"workflow":   "release-agent-os.yml",
			"run_id":     "123456789",
		},
		"artifacts":    artifacts(),
		"images":       images(),
		"checksums":    checksums(),
		"sboms":        references("archives"),
		"signatures":   references("checksums"),
		"provenance":   references("images"),
		"attestations": references("checksums"),
		"homebrew": map[string]any{
			"status":         "publication_blocked",
			"cask_token":     "agent-os",
			"tap_repository": "example/homebrew-tap",
			"commit":         "not-published",
			"path":           "Casks/agent-os.rb",
			"reason":         "public_destination_missing",
		},
		"notes": map[string]any{
			"release_url": "https://github.com/example/agent-os/releases/tag/v0.1.0-rc.1",
			"install":     "install from signed archive, Homebrew cask, or GHCR image",
			"verify":      "verify checksums, signatures, attestations, SBOMs, and provenance",
			"upgrade":     "install the newer immutable tag after signatures, attestations, SBOMs, and provenance validate",
			"rollback":    "install a superseding tag or follow the withdrawal note",
		},
		"rollback": map[string]any{
			"disposition":       "candidate",
			"recovery":          "publish a superseding release",
			"immutable_warning": "release tags are immutable",
		},
	}
}

func artifacts() []any {
	items := make([]any, 0, 4)
	for _, platform := range []struct{ os, arch string }{
		{os: "linux", arch: "amd64"},
		{os: "linux", arch: "arm64"},
		{os: "darwin", arch: "amd64"},
		{os: "darwin", arch: "arm64"},
	} {
		name := "agent-os_v0.1.0-rc.1_" + platform.os + "_" + platform.arch + ".tar.gz"
		items = append(items, map[string]any{
			"name":        name,
			"os":          platform.os,
			"arch":        platform.arch,
			"checksum":    name + ".sha256",
			"signature":   name + ".sigstore.json",
			"sbom":        name + ".spdx.json",
			"attestation": name + ".intoto.jsonl",
		})
	}
	return items
}

func images() []any {
	return []any{
		image("linux/amd64"),
		image("linux/arm64"),
	}
}

func image(platform string) map[string]any {
	return map[string]any{
		"name":        "ghcr.io/example/agent-os",
		"tag":         "v0.1.0-rc.1",
		"platform":    platform,
		"digest":      "sha256:" + strings.Repeat("a", 64),
		"signature":   "ghcr.io/example/agent-os@" + strings.Repeat("b", 64),
		"sbom":        platform + ".spdx.json",
		"provenance":  platform + ".intoto.jsonl",
		"attestation": platform + ".attestation",
	}
}

func checksums() map[string]any {
	return map[string]any{
		"file":        "checksums.txt",
		"algorithm":   "sha256",
		"signature":   "checksums.txt.sigstore.json",
		"attestation": "checksums.txt.intoto.jsonl",
	}
}

func references(subject string) []any {
	return []any{
		map[string]any{
			"subject": subject,
			"uri":     "https://github.com/example/agent-os/releases/download/v0.1.0-rc.1/" + subject,
		},
	}
}

func without(source map[string]any, key string) map[string]any {
	clone := clonePayload(source)
	delete(clone, key)
	return clone
}

func withArtifactField(key string, value any) map[string]any {
	clone := clonePayload(validPayload())
	artifacts := clone["artifacts"].([]any)
	artifacts[0].(map[string]any)[key] = value
	return clone
}

func withImageField(key string, value any) map[string]any {
	clone := clonePayload(validPayload())
	images := clone["images"].([]any)
	images[0].(map[string]any)[key] = value
	return clone
}

func withReferenceField(section string, key string, value any) map[string]any {
	clone := clonePayload(validPayload())
	references := clone[section].([]any)
	references[0].(map[string]any)[key] = value
	return clone
}

func clonePayload(source map[string]any) map[string]any {
	content, err := json.Marshal(source)
	if err != nil {
		panic(err)
	}
	var clone map[string]any
	if err := json.Unmarshal(content, &clone); err != nil {
		panic(err)
	}
	return clone
}

func readSchema(t *testing.T) schemaDocument {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(findRoot(t), "contracts", "release", "v1.schema.json"))
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
	return validator{defs: schema.Defs}.validate("release", value, schema)
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
	if len(schema.Enum) > 0 && !matchesEnum(schema.Enum, value) {
		return fmt.Errorf("%s must match enum", name)
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
		if err := v.validateField(name, key, item, schema); err != nil {
			return err
		}
	}
	return nil
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
	if len(items) < schema.MinItems {
		return fmt.Errorf("%s must contain at least %d items", name, schema.MinItems)
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
	switch value.(type) {
	case int, float64:
		return nil
	default:
		return fmt.Errorf("%s must be an integer", name)
	}
}

func matchesConst(raw json.RawMessage, value any) bool {
	var want any
	if err := json.Unmarshal(raw, &want); err != nil {
		return false
	}
	return reflect.DeepEqual(want, normalized(value))
}

func matchesEnum(items []json.RawMessage, value any) bool {
	for _, item := range items {
		if matchesConst(item, value) {
			return true
		}
	}
	return false
}

func normalized(value any) any {
	content, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var decoded any
	if err := json.Unmarshal(content, &decoded); err != nil {
		return value
	}
	return decoded
}

func findRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if exists(filepath.Join(dir, "go.mod")) && exists(filepath.Join(dir, "quality", "boundary-manifest.json")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("source root not found")
		}
		dir = parent
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
