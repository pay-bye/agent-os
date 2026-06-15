package release_test

import "testing"

func TestSchemaAcceptsCompletePayload(t *testing.T) {
	schema := readSchema(t)

	if err := validate(schema, validPayload()); err != nil {
		t.Fatalf("schema rejected complete payload: %v", err)
	}
}

func TestSchemaRejectsIncompletePayload(t *testing.T) {
	schema := readSchema(t)

	for _, test := range invalidPayloadCases() {
		t.Run(test.name, func(t *testing.T) {
			if err := validate(schema, test.body); err == nil {
				t.Fatal("expected schema rejection")
			}
		})
	}
}

func TestInvalidPayloadCasesUseOpaqueFixtureValues(t *testing.T) {
	cases := invalidPayloadCases()

	requireInvalidArtifactCase(t, cases)
	requireInvalidImageCase(t, cases)
	requireInvalidReferenceCase(t, cases)
}

func TestIdentifierIsRelative(t *testing.T) {
	schema := readSchema(t)

	if schema.ID != "v1.schema.json" {
		t.Fatalf("schema id = %q, want v1.schema.json", schema.ID)
	}
}

func requireInvalidArtifactCase(t *testing.T, cases []payloadCase) {
	t.Helper()

	body := caseBody(t, cases, "invalid artifact enum value")
	artifact := firstArtifact(t, body)
	if artifact["os"] != "opaque" {
		t.Fatalf("artifact fixture value = %v, want opaque", artifact["os"])
	}
}

func requireInvalidImageCase(t *testing.T, cases []payloadCase) {
	t.Helper()

	body := caseBody(t, cases, "invalid image enum value")
	image := firstImage(t, body)
	if image["platform"] != "opaque/platform" {
		t.Fatalf("image fixture value = %v, want opaque/platform", image["platform"])
	}
}

func requireInvalidReferenceCase(t *testing.T, cases []payloadCase) {
	t.Helper()

	body := caseBody(t, cases, "unknown reference property")
	reference := firstReference(t, body, "signatures")
	if reference["extra"] != "value" {
		t.Fatalf("reference fixture value = %v, want value", reference["extra"])
	}
}

func caseBody(t *testing.T, cases []payloadCase, name string) map[string]any {
	t.Helper()

	for _, item := range cases {
		if item.name == name {
			return item.body
		}
	}
	t.Fatalf("case %q not found", name)
	return nil
}

func firstArtifact(t *testing.T, payload map[string]any) map[string]any {
	t.Helper()

	return firstPayloadItem(t, payload, "artifacts")
}

func firstImage(t *testing.T, payload map[string]any) map[string]any {
	t.Helper()

	return firstPayloadItem(t, payload, "images")
}

func firstReference(t *testing.T, payload map[string]any, section string) map[string]any {
	t.Helper()

	return firstPayloadItem(t, payload, section)
}

func firstPayloadItem(t *testing.T, payload map[string]any, section string) map[string]any {
	t.Helper()

	items := payload[section].([]any)
	if len(items) == 0 {
		t.Fatalf("%s must not be empty", section)
	}
	return items[0].(map[string]any)
}
