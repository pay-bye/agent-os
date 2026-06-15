package invocation_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/tests/conformance/schemadoc"

	"gopkg.in/yaml.v3"
)

func TestContractFollowsVersionedOperationSurface(t *testing.T) {
	if err := validateSurface(contractRoot(t)); err != nil {
		t.Fatal(err)
	}
}

func TestContractSurfaceRejectsDrift(t *testing.T) {
	for _, test := range surfaceDriftCases() {
		t.Run(test.name, func(t *testing.T) {
			root := copyContract(t)
			test.edit(t, root)

			err := validateSurface(root)

			if err == nil {
				t.Fatal("expected contract surface rejection")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

type surfaceDriftCase struct {
	name string
	edit func(*testing.T, string)
	want string
}

type textEdit struct {
	path string
	old  string
	new  string
}

type schemaFixture struct {
	name string
	body string
}

type operationSecurityDrift struct {
	operationID string
	value       string
}

type indexDocument struct {
	OpenAPI    string                  `yaml:"openapi"`
	Info       indexInfo               `yaml:"info"`
	Security   []map[string][]string   `yaml:"security"`
	Paths      map[string]pathItems    `yaml:"paths"`
	Components indexComponentsDocument `yaml:"components"`
}

type indexInfo struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
}

type indexComponentsDocument struct {
	SecuritySchemes map[string]schemeDocument   `yaml:"securitySchemes"`
	Responses       map[string]responseDocument `yaml:"responses"`
}

type schemeDocument struct {
	Type   string `yaml:"type"`
	Scheme string `yaml:"scheme"`
}

type pathItems map[string]operationDocument

type operationDocument struct {
	OperationID string                      `yaml:"operationId"`
	Parameters  []parameterDocument         `yaml:"parameters"`
	RequestBody contentDocument             `yaml:"requestBody"`
	Responses   map[string]responseDocument `yaml:"responses"`

	securityPresent bool
}

func (operation *operationDocument) UnmarshalYAML(node *yaml.Node) error {
	var fields struct {
		OperationID string                      `yaml:"operationId"`
		Parameters  []parameterDocument         `yaml:"parameters"`
		RequestBody contentDocument             `yaml:"requestBody"`
		Responses   map[string]responseDocument `yaml:"responses"`
	}
	if err := node.Decode(&fields); err != nil {
		return err
	}
	operation.OperationID = fields.OperationID
	operation.Parameters = fields.Parameters
	operation.RequestBody = fields.RequestBody
	operation.Responses = fields.Responses
	operation.securityPresent = hasMappingKey(node, "security")
	return nil
}

type parameterDocument struct {
	Name     string                  `yaml:"name"`
	Where    string                  `yaml:"in"`
	Required bool                    `yaml:"required"`
	Schema   parameterSchemaDocument `yaml:"schema"`
}

type parameterSchemaDocument struct {
	Type    string   `yaml:"type"`
	Minimum *int     `yaml:"minimum"`
	Maximum *int     `yaml:"maximum"`
	Default any      `yaml:"default"`
	Enum    []string `yaml:"enum"`
}

type responseDocument struct {
	Ref         string                     `yaml:"$ref"`
	Description string                     `yaml:"description"`
	Content     map[string]schemaReference `yaml:"content"`
}

type contentDocument struct {
	Content map[string]schemaReference `yaml:"content"`
}

type schemaReference struct {
	Schema referenceDocument `yaml:"schema"`
}

type referenceDocument struct {
	Ref   string              `yaml:"$ref"`
	Type  string              `yaml:"type"`
	OneOf []referenceDocument `yaml:"oneOf"`
}

type contractSchema struct {
	Schema               string                    `json:"$schema"`
	ID                   string                    `json:"$id"`
	Type                 string                    `json:"type"`
	Properties           map[string]schemaProperty `json:"properties"`
	AdditionalProperties *bool                     `json:"additionalProperties"`
	OneOf                []contractSchema          `json:"oneOf"`
}

type schemaProperty struct {
	Enum []string `json:"enum"`
}

func surfaceDriftCases() []surfaceDriftCase {
	return []surfaceDriftCase{
		{name: "floating version", edit: driftVersion, want: "version"},
		{name: "floating alias", edit: driftFloatingAlias, want: "latest.openapi.yaml"},
		{name: "unknown command path", edit: driftUnknownCommandPath, want: "/inspect"},
		{name: "unknown response code", edit: driftUnknownResponseCode, want: "418"},
		{name: "non-json media type", edit: driftNonJSONMediaType, want: "application/xml"},
		{name: "missing bearer security", edit: driftBearerSecurity, want: "root security"},
		{name: "submit security array override", edit: driftSubmitSecurityArray, want: "operation security"},
		{name: "submit security null override", edit: driftSubmitSecurityNull, want: "operation security"},
		{name: "compatibility security array override", edit: driftCompatibilitySecurityArray, want: "operation security"},
		{name: "compatibility security null override", edit: driftCompatibilitySecurityNull, want: "operation security"},
		{name: "missing unauthorized response", edit: driftUnauthorizedResponse, want: "401"},
		{name: "missing unauthorized component", edit: driftUnauthorizedComponent, want: "component responses"},
		{name: "missing schema reference", edit: driftSchemaReference, want: "missing.request.schema.json"},
		{name: "probe invalid input schema drift", edit: driftProbeInvalidInputSchema, want: "error.response.schema.json"},
		{name: "orphan schema", edit: driftOrphanSchema, want: "orphan.request.schema.json"},
		{name: "schema id drift", edit: driftSchemaID, want: "$id"},
	}
}

func driftVersion(t *testing.T, root string) {
	replaceText(t, root, textEdit{path: "v1.openapi.yaml", old: "version: v1", new: "version: latest"})
}

func driftFloatingAlias(t *testing.T, root string) {
	copyFile(t, filepath.Join(root, "v1.openapi.yaml"), filepath.Join(root, "latest.openapi.yaml"))
}

func driftUnknownCommandPath(t *testing.T, root string) {
	replaceText(t, root, textEdit{
		path: "v1.openapi.yaml",
		old:  "  /submit:",
		new:  "  /inspect:",
	})
}

func driftUnknownResponseCode(t *testing.T, root string) {
	replaceText(t, root, textEdit{
		path: "v1.openapi.yaml",
		old:  `        "409":`,
		new:  `        "418":`,
	})
}

func driftNonJSONMediaType(t *testing.T, root string) {
	replaceText(t, root, textEdit{
		path: "v1.openapi.yaml",
		old:  "          application/json:",
		new:  "          application/xml:",
	})
}

func driftBearerSecurity(t *testing.T, root string) {
	replaceText(t, root, textEdit{
		path: "v1.openapi.yaml",
		old:  "security:\n  - credential: []\n",
		new:  "",
	})
}

func driftSubmitSecurityArray(t *testing.T, root string) {
	driftOperationSecurity(t, root, operationSecurityDrift{operationID: "submit", value: "[]"})
}

func driftSubmitSecurityNull(t *testing.T, root string) {
	driftOperationSecurity(t, root, operationSecurityDrift{operationID: "submit", value: "null"})
}

func driftCompatibilitySecurityArray(t *testing.T, root string) {
	driftOperationSecurity(t, root, operationSecurityDrift{operationID: "compatibility", value: "[]"})
}

func driftCompatibilitySecurityNull(t *testing.T, root string) {
	driftOperationSecurity(t, root, operationSecurityDrift{operationID: "compatibility", value: "null"})
}

func driftOperationSecurity(t *testing.T, root string, drift operationSecurityDrift) {
	replaceText(t, root, textEdit{
		path: "v1.openapi.yaml",
		old:  fmt.Sprintf("      operationId: %s\n", drift.operationID),
		new:  fmt.Sprintf("      operationId: %s\n      security: %s\n", drift.operationID, drift.value),
	})
}

func driftUnauthorizedResponse(t *testing.T, root string) {
	replaceText(t, root, textEdit{
		path: "v1.openapi.yaml",
		old:  "        \"401\":\n          $ref: \"#/components/responses/Unauthorized\"\n",
		new:  "",
	})
}

func driftUnauthorizedComponent(t *testing.T, root string) {
	replaceText(t, root, textEdit{
		path: "v1.openapi.yaml",
		old:  "    Unauthorized:\n      description: Unauthorized\n",
		new:  "",
	})
}

func driftSchemaReference(t *testing.T, root string) {
	replaceText(t, root, textEdit{
		path: "v1.openapi.yaml",
		old:  schemaRef("submit.request.schema.json"),
		new:  "./commands/missing.request.schema.json",
	})
}

func driftProbeInvalidInputSchema(t *testing.T, root string) {
	replaceText(t, root, textEdit{
		path: "v1.openapi.yaml",
		old: `        "400":
          description: invalid_input
          content:
            application/json:
              schema:
                $ref: ./errors/error.response.schema.json
        "401":
          $ref: "#/components/responses/Unauthorized"
  /readyz:`,
		new: `        "400":
          description: invalid_input
          content:
            application/json:
              schema:
                $ref: ./probes/health.response.schema.json
        "401":
          $ref: "#/components/responses/Unauthorized"
  /readyz:`,
	})
}

func driftOrphanSchema(t *testing.T, root string) {
	writeSchema(t, root, schemaFixture{name: "orphan.request.schema.json", body: orphanSchema()})
}

func driftSchemaID(t *testing.T, root string) {
	replaceText(t, root, textEdit{
		path: schemaPath("ack.request.schema.json"),
		old:  `"ack.request.schema.json"`,
		new:  `"ack.request.v2.schema.json"`,
	})
}

func orphanSchema() string {
	return `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "orphan.request.schema.json",
  "type": "object",
  "additionalProperties": false
}`
}

func validateSurface(root string) error {
	if err := validateRoot(root); err != nil {
		return err
	}
	index, err := readIndex(root)
	if err != nil {
		return err
	}
	if err := validateIndex(index); err != nil {
		return err
	}
	schemas, err := readSchemas(root)
	if err != nil {
		return err
	}
	if err := validateSchemaInventory(root, index, schemas); err != nil {
		return err
	}
	if err := validateSchemas(schemas); err != nil {
		return err
	}
	return nil
}

func validateRoot(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return requireExactValues("contract root", names, []string{
		"commands",
		"compatibility",
		"errors",
		"instructions",
		"operations",
		"probes",
		"v1.openapi.yaml",
	})
}

func readIndex(root string) (indexDocument, error) {
	content, err := os.ReadFile(filepath.Join(root, "v1.openapi.yaml"))
	if err != nil {
		return indexDocument{}, err
	}
	var index indexDocument
	if err := yaml.Unmarshal(content, &index); err != nil {
		return indexDocument{}, err
	}
	return index, nil
}

func validateIndex(index indexDocument) error {
	if index.OpenAPI != "3.1.0" {
		return fmt.Errorf("openapi = %q, want 3.1.0", index.OpenAPI)
	}
	if index.Info.Title != "Invocation Contract" {
		return fmt.Errorf("title = %q, want Invocation Contract", index.Info.Title)
	}
	if index.Info.Version != "v1" {
		return fmt.Errorf("version = %q, want v1", index.Info.Version)
	}
	if err := validateSecurity(index); err != nil {
		return err
	}
	if err := requireExactKeys("paths", index.Paths, acceptedPaths()); err != nil {
		return err
	}
	for _, command := range commandNames() {
		if err := validatePath(command, index.Paths["/"+command]); err != nil {
			return err
		}
	}
	for _, probe := range probeNames() {
		if err := validateProbePath(probe, index.Paths["/"+probe]); err != nil {
			return err
		}
	}
	if err := validateMetricsPath(index.Paths["/metrics"]); err != nil {
		return err
	}
	for _, route := range readRoutes() {
		if err := validateReadPath(route, index.Paths[route.path]); err != nil {
			return err
		}
	}
	for _, route := range instructionRoutes() {
		if err := validateInstructionPath(route, index.Paths[route.path]); err != nil {
			return err
		}
	}
	return validateCompatibilityPath(index.Paths["/compatibility"])
}

func validateSecurity(index indexDocument) error {
	if err := validateRootSecurity(index.Security); err != nil {
		return err
	}
	if err := validateSecuritySchemes(index.Components.SecuritySchemes); err != nil {
		return err
	}
	return validateUnauthorizedResponse(index.Components.Responses)
}

func validateRootSecurity(security []map[string][]string) error {
	if len(security) != 1 {
		return fmt.Errorf("root security = %v, want credential bearer requirement", security)
	}
	requirement, ok := security[0]["credential"]
	if !ok || len(requirement) != 0 {
		return fmt.Errorf("root security = %v, want credential bearer requirement", security)
	}
	return nil
}

func validateSecuritySchemes(schemes map[string]schemeDocument) error {
	if err := requireExactKeys("security schemes", schemes, []string{"credential"}); err != nil {
		return err
	}
	scheme := schemes["credential"]
	if scheme.Type != "http" || scheme.Scheme != "bearer" {
		return fmt.Errorf("credential scheme = %s/%s, want http/bearer", scheme.Type, scheme.Scheme)
	}
	return nil
}

func validateUnauthorizedResponse(responses map[string]responseDocument) error {
	if err := requireExactKeys("component responses", responses, []string{"Unauthorized"}); err != nil {
		return err
	}
	response := responses["Unauthorized"]
	if response.Description != "Unauthorized" {
		return fmt.Errorf("Unauthorized response description = %q, want Unauthorized", response.Description)
	}
	if len(response.Content) != 0 {
		return fmt.Errorf("Unauthorized response must not declare content")
	}
	return nil
}

func validatePath(command string, methods pathItems) error {
	if err := requireExactKeys("/"+command+" methods", methods, []string{"post"}); err != nil {
		return err
	}
	operation := methods["post"]
	if operation.OperationID != command {
		return fmt.Errorf("%s operationId = %q, want %q", command, operation.OperationID, command)
	}
	if err := validateOperationSecurity(command, operation); err != nil {
		return err
	}
	if err := validateRequest(command, operation.RequestBody); err != nil {
		return err
	}
	return validateResponses(command, operation.Responses)
}

func validateOperationSecurity(name string, operation operationDocument) error {
	if operation.securityPresent {
		return fmt.Errorf("%s operation security override must be absent", name)
	}
	return nil
}

func validateRequest(command string, body contentDocument) error {
	ref, err := requireJSONRef(command+" request", body.Content)
	if err != nil {
		return err
	}
	return requireRef(command+" request", ref, schemaRef(command+".request.schema.json"))
}

func validateResponses(command string, responses map[string]responseDocument) error {
	if err := requireExactKeys(command+" responses", responses, acceptedResponseCodes()); err != nil {
		return err
	}
	for _, code := range acceptedResponseCodes() {
		if err := validateResponse(command, code, responses[code]); err != nil {
			return err
		}
	}
	return nil
}

func validateResponse(command string, code string, response responseDocument) error {
	if code == "401" {
		if err := requireRef(command+" response "+code, response.Ref, "#/components/responses/Unauthorized"); err != nil {
			return err
		}
		if len(response.Content) != 0 {
			return fmt.Errorf("%s response %s must not declare content", command, code)
		}
		return nil
	}
	ref, err := requireJSONRef(command+" response "+code, response.Content)
	if err != nil {
		return err
	}
	if code == "200" {
		return requireRef(command+" response "+code, ref, schemaRef(command+".response.schema.json"))
	}
	return requireRef(command+" response "+code, ref, schemaRef("error.response.schema.json"))
}

func validateCompatibilityPath(methods pathItems) error {
	if err := requireExactKeys("/compatibility methods", methods, []string{"get"}); err != nil {
		return err
	}
	operation := methods["get"]
	if operation.OperationID != "compatibility" {
		return fmt.Errorf("compatibility operationId = %q, want compatibility", operation.OperationID)
	}
	if err := validateOperationSecurity("compatibility", operation); err != nil {
		return err
	}
	if len(operation.RequestBody.Content) > 0 {
		return fmt.Errorf("compatibility request body must be absent")
	}
	if err := requireExactKeys("compatibility responses", operation.Responses, []string{"200", "401"}); err != nil {
		return err
	}
	if err := validateResponse("compatibility", "401", operation.Responses["401"]); err != nil {
		return err
	}
	ref, err := requireJSONRef("compatibility response 200", operation.Responses["200"].Content)
	if err != nil {
		return err
	}
	return requireRef("compatibility response 200", ref, schemaRef("compatibility.response.schema.json"))
}

func validateProbePath(probe string, methods pathItems) error {
	if err := requireExactKeys("/"+probe+" methods", methods, []string{"get"}); err != nil {
		return err
	}
	operation := methods["get"]
	if operation.OperationID != probe {
		return fmt.Errorf("%s operationId = %q, want %q", probe, operation.OperationID, probe)
	}
	if err := validateOperationSecurity(probe, operation); err != nil {
		return err
	}
	if len(operation.RequestBody.Content) > 0 {
		return fmt.Errorf("%s request body must be absent", probe)
	}
	for _, code := range probeResponseCodes(probe) {
		if err := validateProbeResponse(probe, code, operation.Responses[code]); err != nil {
			return err
		}
	}
	return requireExactKeys(probe+" responses", operation.Responses, probeResponseCodes(probe))
}

func validateProbeResponse(probe string, code string, response responseDocument) error {
	if code == "401" {
		return validateResponse(probe, code, response)
	}
	ref, err := requireJSONRef(probe+" response "+code, response.Content)
	if err != nil {
		return err
	}
	return requireRef(probe+" response "+code, ref, probeResponseSchema(probe, code))
}

func probeResponseSchema(probe string, code string) string {
	if code == "400" {
		return schemaRef("error.response.schema.json")
	}
	return schemaRef(probe + ".response.schema.json")
}

func probeResponseCodes(probe string) []string {
	if probe == "readyz" {
		return []string{"200", "400", "401", "503"}
	}
	return []string{"200", "400", "401"}
}

func validateMetricsPath(methods pathItems) error {
	if err := requireExactKeys("/metrics methods", methods, []string{"get"}); err != nil {
		return err
	}
	operation := methods["get"]
	if operation.OperationID != "metrics" {
		return fmt.Errorf("metrics operationId = %q, want metrics", operation.OperationID)
	}
	if err := validateOperationSecurity("metrics", operation); err != nil {
		return err
	}
	if len(operation.RequestBody.Content) > 0 {
		return fmt.Errorf("metrics request body must be absent")
	}
	if err := validateMetricsResponse(operation.Responses["200"]); err != nil {
		return err
	}
	if err := validateResponse("metrics", "401", operation.Responses["401"]); err != nil {
		return err
	}
	return requireExactKeys("metrics responses", operation.Responses, []string{"200", "401"})
}

func validateMetricsResponse(response responseDocument) error {
	contentType := "text/plain; version=0.0.4; charset=utf-8"
	if err := requireExactKeys("metrics response 200 content", response.Content, []string{contentType}); err != nil {
		return err
	}
	schema := response.Content[contentType].Schema
	if schema.Type != "string" {
		return fmt.Errorf("metrics response 200 schema type = %q, want string", schema.Type)
	}
	if schema.Ref != "" {
		return fmt.Errorf("metrics response 200 schema ref = %q, want none", schema.Ref)
	}
	return nil
}

func validateReadPath(route readRoute, methods pathItems) error {
	if err := requireExactKeys(route.path+" methods", methods, []string{"get"}); err != nil {
		return err
	}
	operation := methods["get"]
	if operation.OperationID != route.operation {
		return fmt.Errorf("%s operationId = %q, want %q", route.path, operation.OperationID, route.operation)
	}
	if err := validateOperationSecurity(route.operation, operation); err != nil {
		return err
	}
	if len(operation.RequestBody.Content) > 0 {
		return fmt.Errorf("%s request body must be absent", route.path)
	}
	if err := validateReadParameters(route, operation.Parameters); err != nil {
		return err
	}
	if route.aggregate {
		return validateAggregateReadResponses(route, operation.Responses)
	}
	return validateDedicatedReadResponses(route, operation.Responses)
}

func validateReadParameters(route readRoute, parameters []parameterDocument) error {
	got := parametersByKey(parameters)
	want := parameterSpecKeys(route.parameters)
	if err := requireExactKeys(route.operation+" parameters", got, want); err != nil {
		return err
	}
	for _, spec := range route.parameters {
		if err := validateParameter(route.operation, got[parameterKey(spec.location, spec.name)], spec); err != nil {
			return err
		}
	}
	return nil
}

func validateParameter(operation string, parameter parameterDocument, spec parameterSpec) error {
	if parameter.Name != spec.name {
		return fmt.Errorf("%s parameter name = %q, want %q", operation, parameter.Name, spec.name)
	}
	if parameter.Where != spec.location {
		return fmt.Errorf("%s parameter %s in = %q, want %q", operation, spec.name, parameter.Where, spec.location)
	}
	if parameter.Required != spec.required {
		return fmt.Errorf("%s parameter %s required = %t, want %t", operation, spec.name, parameter.Required, spec.required)
	}
	if parameter.Schema.Type != spec.schemaType {
		return fmt.Errorf("%s parameter %s type = %q, want %q", operation, spec.name, parameter.Schema.Type, spec.schemaType)
	}
	if err := validateParameterBounds(operation, parameter, spec); err != nil {
		return err
	}
	if err := validateParameterDefault(operation, parameter, spec); err != nil {
		return err
	}
	return validateParameterEnum(operation, parameter, spec)
}

func validateParameterBounds(operation string, parameter parameterDocument, spec parameterSpec) error {
	if intValue(parameter.Schema.Minimum) != intValue(spec.minimum) {
		return fmt.Errorf("%s parameter %s minimum = %v, want %v", operation, spec.name, parameter.Schema.Minimum, spec.minimum)
	}
	if intValue(parameter.Schema.Maximum) != intValue(spec.maximum) {
		return fmt.Errorf("%s parameter %s maximum = %v, want %v", operation, spec.name, parameter.Schema.Maximum, spec.maximum)
	}
	return nil
}

func validateParameterDefault(operation string, parameter parameterDocument, spec parameterSpec) error {
	if spec.defaultValue == "" {
		if parameter.Schema.Default != nil {
			return fmt.Errorf("%s parameter %s default = %v, want absent", operation, spec.name, parameter.Schema.Default)
		}
		return nil
	}
	if fmt.Sprint(parameter.Schema.Default) != spec.defaultValue {
		return fmt.Errorf("%s parameter %s default = %v, want %s", operation, spec.name, parameter.Schema.Default, spec.defaultValue)
	}
	return nil
}

func validateParameterEnum(operation string, parameter parameterDocument, spec parameterSpec) error {
	if len(spec.enum) == 0 {
		if len(parameter.Schema.Enum) != 0 {
			return fmt.Errorf("%s parameter %s enum = %v, want absent", operation, spec.name, parameter.Schema.Enum)
		}
		return nil
	}
	return requireExactValues(operation+" parameter "+spec.name+" enum", parameter.Schema.Enum, spec.enum)
}

func parametersByKey(parameters []parameterDocument) map[string]parameterDocument {
	items := map[string]parameterDocument{}
	for _, parameter := range parameters {
		items[parameterKey(parameter.Where, parameter.Name)] = parameter
	}
	return items
}

func parameterSpecKeys(parameters []parameterSpec) []string {
	keys := make([]string, 0, len(parameters))
	for _, parameter := range parameters {
		keys = append(keys, parameterKey(parameter.location, parameter.name))
	}
	return keys
}

func parameterKey(location string, name string) string {
	return location + ":" + name
}

func intValue(value *int) int {
	if value == nil {
		return -1
	}
	return *value
}

func validateAggregateReadResponses(route readRoute, responses map[string]responseDocument) error {
	if err := requireExactKeys(route.operation+" responses", responses, []string{"200", "400", "401", "503"}); err != nil {
		return err
	}
	if err := validateResponse(route.operation, "400", responses["400"]); err != nil {
		return err
	}
	if err := validateResponse(route.operation, "401", responses["401"]); err != nil {
		return err
	}
	for _, code := range []string{"200", "503"} {
		ref, err := requireJSONRef(route.operation+" response "+code, responses[code].Content)
		if err != nil {
			return err
		}
		if err := requireRef(route.operation+" response "+code, ref, schemaRef(route.schema)); err != nil {
			return err
		}
	}
	return nil
}

func validateDedicatedReadResponses(route readRoute, responses map[string]responseDocument) error {
	if err := requireExactKeys(route.operation+" responses", responses, []string{"200", "400", "401", "409"}); err != nil {
		return err
	}
	for _, code := range []string{"400", "409"} {
		if err := validateResponse(route.operation, code, responses[code]); err != nil {
			return err
		}
	}
	if err := validateResponse(route.operation, "401", responses["401"]); err != nil {
		return err
	}
	ref, err := requireJSONRef(route.operation+" response 200", responses["200"].Content)
	if err != nil {
		return err
	}
	return requireRef(route.operation+" response 200", ref, schemaRef(route.schema))
}

func validateInstructionPath(route instructionRoute, methods pathItems) error {
	if err := requireExactKeys(route.path+" methods", methods, []string{"post"}); err != nil {
		return err
	}
	operation := methods["post"]
	if operation.OperationID != route.operation {
		return fmt.Errorf("%s operationId = %q, want %q", route.path, operation.OperationID, route.operation)
	}
	if err := validateOperationSecurity(route.operation, operation); err != nil {
		return err
	}
	if err := validateInstructionParameters(route.operation, operation.Parameters); err != nil {
		return err
	}
	if err := validateInstructionRequest(route, operation.RequestBody); err != nil {
		return err
	}
	return validateInstructionResponses(route.operation, operation.Responses)
}

func validateInstructionParameters(operation string, parameters []parameterDocument) error {
	got := parametersByKey(parameters)
	key := parameterKey("header", "Operator-Key")
	if err := requireExactKeys(operation+" parameters", got, []string{key}); err != nil {
		return err
	}
	return validateParameter(operation, got[key], parameterSpec{
		name:       "Operator-Key",
		location:   "header",
		required:   true,
		schemaType: "string",
	})
}

func validateInstructionRequest(route instructionRoute, body contentDocument) error {
	ref, err := requireJSONRef(route.operation+" request", body.Content)
	if err != nil {
		return err
	}
	return requireRef(route.operation+" request", ref, schemaRef(route.request))
}

func validateInstructionResponses(operation string, responses map[string]responseDocument) error {
	if err := requireExactKeys(operation+" responses", responses, []string{"200", "400", "401", "409"}); err != nil {
		return err
	}
	if err := validateResponse(operation, "400", responses["400"]); err != nil {
		return err
	}
	if err := validateResponse(operation, "401", responses["401"]); err != nil {
		return err
	}
	ref, err := requireJSONRef(operation+" response 200", responses["200"].Content)
	if err != nil {
		return err
	}
	if err := requireRef(operation+" response 200", ref, schemaRef("instruction.response.schema.json")); err != nil {
		return err
	}
	return validateInstructionConflictResponse(operation, responses["409"])
}

func validateInstructionConflictResponse(operation string, response responseDocument) error {
	refs, err := requireJSONRefs(operation+" response 409", response.Content)
	if err != nil {
		return err
	}
	return requireExactValues(operation+" response 409 refs", refs, []string{
		schemaRef("instruction.response.schema.json"),
		schemaRef("error.response.schema.json"),
	})
}

func readSchemas(root string) (map[string]contractSchema, error) {
	schemas := map[string]contractSchema{}
	for _, family := range schemaFamilies() {
		entries, err := os.ReadDir(filepath.Join(root, family))
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				return nil, fmt.Errorf("%s contains subdirectory %s", family, entry.Name())
			}
			if _, ok := schemas[entry.Name()]; ok {
				return nil, fmt.Errorf("schema %s appears in multiple families", entry.Name())
			}
			schema, err := readContractSchema(root, filepath.Join(family, entry.Name()))
			if err != nil {
				return nil, err
			}
			schemas[entry.Name()] = schema
		}
	}
	return schemas, nil
}

func readContractSchema(root string, path string) (contractSchema, error) {
	content, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		return contractSchema{}, err
	}
	var schema contractSchema
	if err := json.Unmarshal(content, &schema); err != nil {
		return contractSchema{}, err
	}
	return schema, nil
}

func validateSchemaInventory(
	root string,
	index indexDocument,
	schemas map[string]contractSchema,
) error {
	if err := requireExactKeys("schemas", schemas, acceptedSchemaNames()); err != nil {
		return err
	}
	refs, err := schemaRefs(index)
	if err != nil {
		return err
	}
	for _, name := range acceptedSchemaNames() {
		if !refs[name] {
			return fmt.Errorf("schema %s is not referenced", name)
		}
		if !exists(filepath.Join(root, schemaPath(name))) {
			return fmt.Errorf("schema %s is missing", name)
		}
	}
	return nil
}

func schemaRefs(index indexDocument) (map[string]bool, error) {
	refs := map[string]bool{}
	for _, command := range commandNames() {
		operation := index.Paths["/"+command]["post"]
		if err := addRef(refs, operation.RequestBody.Content["application/json"].Schema.Ref); err != nil {
			return nil, err
		}
		for _, response := range operation.Responses {
			if response.Ref != "" {
				continue
			}
			if err := addSchemaRefs(refs, response.Content["application/json"].Schema); err != nil {
				return nil, err
			}
		}
	}
	operation := index.Paths["/compatibility"]["get"]
	if err := addSchemaRefs(refs, operation.Responses["200"].Content["application/json"].Schema); err != nil {
		return nil, err
	}
	for _, route := range readRoutes() {
		operation := index.Paths[route.path]["get"]
		for _, response := range operation.Responses {
			if response.Ref != "" {
				continue
			}
			if err := addSchemaRefs(refs, response.Content["application/json"].Schema); err != nil {
				return nil, err
			}
		}
	}
	for _, route := range instructionRoutes() {
		operation := index.Paths[route.path]["post"]
		if err := addSchemaRefs(refs, operation.RequestBody.Content["application/json"].Schema); err != nil {
			return nil, err
		}
		for _, response := range operation.Responses {
			if response.Ref != "" {
				continue
			}
			if err := addSchemaRefs(refs, response.Content["application/json"].Schema); err != nil {
				return nil, err
			}
		}
	}
	for _, probe := range probeNames() {
		for _, response := range index.Paths["/"+probe]["get"].Responses {
			if response.Ref != "" {
				continue
			}
			if err := addSchemaRefs(refs, response.Content["application/json"].Schema); err != nil {
				return nil, err
			}
		}
	}
	return refs, nil
}

func addRef(refs map[string]bool, ref string) error {
	name, ok := schemaNameByRef(ref)
	if !ok {
		return fmt.Errorf("schema ref %q is not accepted", ref)
	}
	refs[name] = true
	return nil
}

func addSchemaRefs(refs map[string]bool, schema referenceDocument) error {
	names, err := schemaRefsFrom(schema)
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := addRef(refs, name); err != nil {
			return err
		}
	}
	return nil
}

func schemaRefsFrom(schema referenceDocument) ([]string, error) {
	if schema.Ref != "" {
		return []string{schema.Ref}, nil
	}
	if len(schema.OneOf) == 0 {
		return nil, fmt.Errorf("schema reference is missing $ref")
	}
	refs := make([]string, 0, len(schema.OneOf))
	for _, item := range schema.OneOf {
		values, err := schemaRefsFrom(item)
		if err != nil {
			return nil, err
		}
		refs = append(refs, values...)
	}
	return refs, nil
}

func validateSchemas(schemas map[string]contractSchema) error {
	for _, name := range acceptedSchemaNames() {
		if err := validateSchema(name, schemas[name]); err != nil {
			return err
		}
	}
	return validateErrorSchema(schemas["error.response.schema.json"])
}

func validateSchema(name string, schema contractSchema) error {
	if schema.Schema != "https://json-schema.org/draft/2020-12/schema" {
		return fmt.Errorf("%s $schema = %q", name, schema.Schema)
	}
	if schema.ID != name {
		return fmt.Errorf("%s $id = %q", name, schema.ID)
	}
	return validateStrictRoot(name, schema)
}

func validateStrictRoot(name string, schema contractSchema) error {
	if len(schema.OneOf) == 0 {
		return validateStrictObject(name, schema)
	}
	for index, variant := range schema.OneOf {
		if err := validateStrictObject(fmt.Sprintf("%s oneOf[%d]", name, index), variant); err != nil {
			return err
		}
	}
	return nil
}

func validateStrictObject(name string, schema contractSchema) error {
	if schema.Type != "object" {
		return fmt.Errorf("%s root type = %q, want object", name, schema.Type)
	}
	if schema.AdditionalProperties == nil || *schema.AdditionalProperties {
		return fmt.Errorf("%s must set additionalProperties false", name)
	}
	return nil
}

func validateErrorSchema(schema contractSchema) error {
	return requireExactValues("error enum", schema.Properties["error"].Enum, behaviorErrorNames())
}

func requireJSONRef(name string, content map[string]schemaReference) (string, error) {
	if err := requireExactKeys(name+" content", content, []string{"application/json"}); err != nil {
		return "", err
	}
	return content["application/json"].Schema.Ref, nil
}

func requireJSONRefs(name string, content map[string]schemaReference) ([]string, error) {
	if err := requireExactKeys(name+" content", content, []string{"application/json"}); err != nil {
		return nil, err
	}
	return schemaRefsFrom(content["application/json"].Schema)
}

func requireRef(name string, got string, want string) error {
	if got != want {
		return fmt.Errorf("%s ref = %q, want %q", name, got, want)
	}
	return nil
}

func hasMappingKey(node *yaml.Node, key string) bool {
	if node.Kind != yaml.MappingNode {
		return false
	}
	for index := 0; index < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			return true
		}
	}
	return false
}

func requireExactKeys[V any](name string, items map[string]V, want []string) error {
	return requireExactValues(name, mapKeys(items), want)
}

func requireExactValues(name string, got []string, want []string) error {
	slices.Sort(got)
	want = slices.Clone(want)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		return fmt.Errorf("%s = %v, want %v", name, got, want)
	}
	return nil
}

func mapKeys[V any](items map[string]V) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	return keys
}

func acceptedPaths() []string {
	paths := make([]string, 0, len(commandNames()))
	for _, command := range commandNames() {
		paths = append(paths, "/"+command)
	}
	for _, probe := range probeNames() {
		paths = append(paths, "/"+probe)
	}
	for _, name := range metricRouteNames() {
		paths = append(paths, "/"+name)
	}
	for _, route := range readRoutes() {
		paths = append(paths, route.path)
	}
	for _, route := range instructionRoutes() {
		paths = append(paths, route.path)
	}
	paths = append(paths, "/compatibility")
	return paths
}

func acceptedResponseCodes() []string {
	return []string{"200", "400", "401", "404", "409"}
}

func acceptedSchemaNames() []string {
	names := make([]string, 0, len(acceptedSchemas()))
	for _, schema := range acceptedSchemas() {
		names = append(names, schema.name)
	}
	return names
}

func schemaPath(name string) string {
	for _, schema := range acceptedSchemas() {
		if schema.name == name {
			return filepath.Join(schema.family, name)
		}
	}
	return filepath.Join("unknown", name)
}

func schemaRef(name string) string {
	return "./" + filepath.ToSlash(schemaPath(name))
}

func schemaNameByRef(ref string) (string, bool) {
	for _, schema := range acceptedSchemas() {
		if schemaRef(schema.name) == ref {
			return schema.name, true
		}
	}
	return "", false
}

func schemaFamilies() []string {
	return []string{"commands", "compatibility", "errors", "instructions", "operations", "probes"}
}

type acceptedSchema struct {
	name   string
	family string
}

func acceptedSchemas() []acceptedSchema {
	schemas := []acceptedSchema{
		{name: "compatibility.response.schema.json", family: "compatibility"},
		{name: "error.response.schema.json", family: "errors"},
	}
	for _, command := range commandNames() {
		schemas = append(
			schemas,
			acceptedSchema{name: command + ".request.schema.json", family: "commands"},
			acceptedSchema{name: command + ".response.schema.json", family: "commands"},
		)
	}
	for _, probe := range probeNames() {
		schemas = append(schemas, acceptedSchema{name: probe + ".response.schema.json", family: "probes"})
	}
	for _, route := range readRoutes() {
		schemas = append(schemas, acceptedSchema{name: route.schema, family: "operations"})
	}
	for _, route := range instructionRoutes() {
		schemas = append(schemas, acceptedSchema{name: route.request, family: "instructions"})
	}
	schemas = append(schemas, acceptedSchema{name: "instruction.response.schema.json", family: "instructions"})
	return schemas
}

func copyContract(t *testing.T) string {
	t.Helper()

	source := contractRoot(t)
	target := filepath.Join(t.TempDir(), "invocation")
	paths, err := filesUnder(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		relative, err := filepath.Rel(source, path)
		if err != nil {
			t.Fatal(err)
		}
		copyFile(t, path, filepath.Join(target, relative))
	}
	return target
}

func copyFile(t *testing.T, source string, target string) {
	t.Helper()

	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, content, 0o600); err != nil {
		t.Fatal(err)
	}
}

func replaceText(t *testing.T, root string, edit textEdit) {
	t.Helper()

	path := filepath.Join(root, edit.path)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(content), edit.old, edit.new, 1)
	if updated == string(content) {
		t.Fatalf("%s did not contain %q", edit.path, edit.old)
	}
	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeSchema(t *testing.T, root string, fixture schemaFixture) {
	t.Helper()

	path := filepath.Join(root, "commands", fixture.name)
	if err := os.WriteFile(path, []byte(fixture.body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func filesUnder(root string) ([]string, error) {
	return schemadoc.FilesUnder(root)
}
