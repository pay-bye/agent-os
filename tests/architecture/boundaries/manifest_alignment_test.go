package checks

import "testing"

func TestRejectsInvalidPolicy(t *testing.T) {
	root := fixture(t, map[string]string{
		"quality/boundary-manifest.json": `{"schema_version":2}`,
	})

	_, err := loadPolicy(root)

	requireErrorContains(t, err, "boundary manifest invalid")
}

func TestRejectsForbiddenImportWithoutSource(t *testing.T) {
	assertInvalidPolicy(t, manifestWithForbiddenImport(`"to": ["github.com/pay-bye/agent-os/internal/model"]`))
}

func TestRejectsForbiddenImportWithoutTarget(t *testing.T) {
	assertInvalidPolicy(t, manifestWithForbiddenImport(`"from": ["internal/flow"]`))
}

func TestRejectsForbiddenImportWithBlankSource(t *testing.T) {
	assertInvalidPolicy(t, manifestWithForbiddenImport(`
		"from": [""],
		"to": ["github.com/pay-bye/agent-os/internal/model"]
	`))
}

func TestRejectsForbiddenImportWithBlankTarget(t *testing.T) {
	assertInvalidPolicy(t, manifestWithForbiddenImport(`
		"from": ["internal/flow"],
		"to": [""]
	`))
}

func TestRejectsForbiddenImportEndpointOutsideToolchainGraph(t *testing.T) {
	assertInvalidPolicyWithFiles(t, manifestWithForbiddenImport(`
		"from": ["internal/flow"],
		"to": ["github.com/pay-bye/agent-os/internal/model"]
	`), map[string]string{
		"internal/flow/file.go":       "package flow\n",
		"internal/model/file_test.go": "package model\n",
	})
}

func TestCurrentTreeConforms(t *testing.T) {
	root := findRoot(t)
	policy, err := loadPolicy(root)
	if err != nil {
		t.Fatal(err)
	}

	violations, err := scanTree(root, policy)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("expected current tree to conform, got %v", violations)
	}
}

func assertInvalidPolicy(t *testing.T, manifest string) {
	t.Helper()

	assertInvalidPolicyWithFiles(t, manifest, realPackageFiles())
}

func assertInvalidPolicyWithFiles(t *testing.T, manifest string, files map[string]string) {
	t.Helper()

	files["go.mod"] = "module github.com/pay-bye/agent-os\n"
	files["quality/boundary-manifest.json"] = manifest
	root := fixture(t, files)

	_, err := loadPolicy(root)

	requireErrorContains(t, err, "boundary manifest invalid")
}

func realPackageFiles() map[string]string {
	return map[string]string{
		"internal/flow/file.go":  "package flow\n",
		"internal/model/file.go": "package model\n",
	}
}

func manifestWithForbiddenImport(fields string) string {
	return `{
		"schema_version": 1,
		"root": ".",
		"allowed_top_level_roots": {
			"rule": "top-level-source-root-category-preservation",
			"names": ["go.mod", "internal", "quality"],
			"message": "top-level source roots are limited to the declared categories"
		},
		"internal_packages": [
			"internal/flow",
			"internal/model"
		],
		"forbidden_imports": [
			{
				"rule": "model-import-boundary",
				"message": "flow package must not import model package",
				` + fields + `
			}
		]
	}`
}
