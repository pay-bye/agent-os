package checks

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestArchitectureEvidenceIsSortedByResponsibility(t *testing.T) {
	assertDirectoryContents(t, "tests/architecture", []string{
		"boundaries",
		"release",
		"verification",
	})
	assertDirectoryContents(t, "tests/architecture/boundaries", []string{
		"boundary_fixture_test.go",
		"directory_topology_test.go",
		"manifest_alignment_test.go",
		"package_graph_test.go",
		"package_import_test.go",
		"response_shape_test.go",
	})
	assertDirectoryContents(t, "tests/architecture/release", []string{
		"release_contract_test.go",
		"release_evidence_test.go",
		"release_fixture_test.go",
		"release_verify_test.go",
	})
	assertDirectoryContents(t, "tests/architecture/verification", []string{
		"coverage_test.go",
		"protected_paths_test.go",
		"verify_command_test.go",
		"verify_fixture_test.go",
	})
}

func TestRejectsUnauthorizedRoot(t *testing.T) {
	assertViolation(t, map[string]string{
		"tmp/file.txt": "scratch",
	}, "top-level-source-root-category-preservation", "tmp")
}

func TestRejectsForbiddenPaths(t *testing.T) {
	policy := mustLoadPolicy(t)

	for _, rule := range policy.ForbiddenPaths {
		t.Run(rule.Rule, func(t *testing.T) {
			path := pathMatching(rule.Pattern)
			assertViolation(t, map[string]string{path: "x"}, rule.Rule, path)
		})
	}
}

func TestContractDocumentsStayUnderContracts(t *testing.T) {
	assertViolation(t, map[string]string{
		"internal/sample/spec.schema.json": "{}",
	}, "contract-in-internal", "internal/sample/spec.schema.json")

	assertClean(t, map[string]string{
		"contracts/spec.schema.json": "{}",
	})
}

func TestMigrationFilesStayUnderStorage(t *testing.T) {
	assertViolation(t, map[string]string{
		"internal/sample/migrations/001.sql": "select 1;",
	}, "migration-outside-storage", "internal/sample/migrations/001.sql")

	assertClean(t, map[string]string{
		"internal/storage/postgres/migrations/001.sql": "select 1;",
	})
}

func TestRejectsTransportWithoutAcceptedProtocol(t *testing.T) {
	assertViolation(t, map[string]string{
		"internal/transport/smtp/file.go": "package smtp",
	}, "transport-non-protocol-subdir", "internal/transport/smtp")
}

func TestAcceptsConfiguredTransportProtocol(t *testing.T) {
	assertClean(t, map[string]string{
		"internal/transport/http/file.go": "package http",
	})
}

func assertDirectoryContents(t *testing.T, relativePath string, want []string) {
	t.Helper()

	got := directoryContents(t, relativePath)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s entries = %v, want %v", relativePath, got, want)
	}
}

func directoryContents(t *testing.T, relativePath string) []string {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(findRoot(t), filepath.FromSlash(relativePath)))
	if err != nil {
		t.Fatal(err)
	}
	items := make([]string, 0, len(entries))
	for _, entry := range entries {
		items = append(items, entry.Name())
	}
	return items
}

func pathMatching(pattern string) string {
	item := expandPattern(pattern)[0]
	segments := splitPath(item)
	for index, segment := range segments {
		switch {
		case segment == "**":
			segments[index] = "sample"
		case containsPatternSyntax(segment):
			segments[index] = "sample"
		}
	}
	return joinPath(segments)
}
