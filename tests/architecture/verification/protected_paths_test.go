package checks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnitRejectsMissingProtectedPaths(t *testing.T) {
	root := cleanRoot(t)
	installCommands(t, root, commandBehavior{})
	if err := os.Remove(filepath.Join(root, "quality", "protected-paths.json")); err != nil {
		t.Fatal(err)
	}

	output, err := runScript(t, root, "--unit")

	requireCommandFailure(t, err)
	requireContains(t, output, "protected paths missing: quality/protected-paths.json")
}

func TestUnitRejectsImmutableProtectedPathChanges(t *testing.T) {
	for _, test := range immutableProtectedPathCases() {
		assertProtectedPathFailure(t, test)
	}
}

func TestUnitRejectsAppendOnlyProtectedPathChanges(t *testing.T) {
	for _, test := range appendOnlyProtectedPathCases() {
		assertProtectedPathFailure(t, test)
	}
}

func TestUnitAcceptsAppendOnlyAdd(t *testing.T) {
	root := cleanRoot(t)
	installCommands(t, root, commandBehavior{GitStatus: "?? log.txt"})
	writeFile(t, root, "quality/protected-paths.json", protectedManifest("append_only", "log.txt"))

	output, err := runScript(t, root, "--unit")

	if err != nil {
		t.Fatalf("expected append-only add to pass, got %v\n%s", err, output)
	}
}

func TestUnitRejectsInvalidProtectedPathManifest(t *testing.T) {
	cases := []struct {
		name     string
		manifest string
	}{
		{name: "wrong version", manifest: `{"schema_version":2,"path_root":"source_root","protections":[]}`},
		{name: "wrong root", manifest: `{"schema_version":1,"path_root":"repository","protections":[]}`},
		{name: "unknown mode", manifest: `{"schema_version":1,"path_root":"source_root","protections":[{"mode":"edit","patterns":["x"],"reason":"fixture"}]}`},
		{name: "missing reason", manifest: `{"schema_version":1,"path_root":"source_root","protections":[{"mode":"immutable","patterns":["x"],"reason":""}]}`},
		{name: "absolute path", manifest: `{"schema_version":1,"path_root":"source_root","protections":[{"mode":"immutable","patterns":["/outside/x"],"reason":"fixture"}]}`},
		{name: "empty path segment", manifest: `{"schema_version":1,"path_root":"source_root","protections":[{"mode":"immutable","patterns":["quality//x"],"reason":"fixture"}]}`},
		{name: "dot path segment", manifest: `{"schema_version":1,"path_root":"source_root","protections":[{"mode":"immutable","patterns":["quality/./x"],"reason":"fixture"}]}`},
		{name: "direct traversal", manifest: `{"schema_version":1,"path_root":"source_root","protections":[{"mode":"immutable","patterns":["../outside/x"],"reason":"fixture"}]}`},
		{name: "nested traversal", manifest: `{"schema_version":1,"path_root":"source_root","protections":[{"mode":"immutable","patterns":["quality/../../outside/x"],"reason":"fixture"}]}`},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			root := cleanRoot(t)
			installCommands(t, root, commandBehavior{})
			writeFile(t, root, "quality/protected-paths.json", test.manifest)

			output, err := runScript(t, root, "--unit")

			requireCommandFailure(t, err)
			requireContains(t, output, "protected paths invalid")
		})
	}
}

type protectedPathCase struct {
	name     string
	manifest string
	status   string
	message  string
}

func immutableProtectedPathCases() []protectedPathCase {
	return []protectedPathCase{
		immutableProtectedPathCase("added", "?? locked.txt"),
		immutableProtectedPathCase("modified", " M locked.txt"),
		immutableProtectedPathCase("deleted", " D locked.txt"),
		immutableProtectedPathCase("renamed", "R  old.txt -> locked.txt"),
		immutableProtectedPathCase("copied", "C  old.txt -> locked.txt"),
		immutableProtectedPathCase("typed", " T locked.txt"),
		immutableProtectedPathCase("unmerged", "UU locked.txt"),
		immutableProtectedPathCase("unknown", "!! locked.txt"),
	}
}

func appendOnlyProtectedPathCases() []protectedPathCase {
	return []protectedPathCase{
		appendOnlyProtectedPathCase("modified", " M log.txt", "M"),
		appendOnlyProtectedPathCase("deleted", " D log.txt", "D"),
		appendOnlyProtectedPathCase("renamed into path", "R  old.txt -> log.txt", "R"),
		appendOnlyProtectedPathCase("copied", "C  old.txt -> log.txt", "C"),
		appendOnlyProtectedPathCase("typed", " T log.txt", "T"),
		appendOnlyProtectedPathCase("unmerged", "UU log.txt", "U"),
		appendOnlyProtectedPathCase("unknown", "!! log.txt", "?"),
	}
}

func immutableProtectedPathCase(name string, status string) protectedPathCase {
	return protectedPathCase{
		name:     "immutable " + name,
		manifest: protectedManifest("immutable", "locked.txt"),
		status:   status,
		message:  "protected path changed: mode=immutable path=locked.txt",
	}
}

func appendOnlyProtectedPathCase(name string, status string, diagnostic string) protectedPathCase {
	return protectedPathCase{
		name:     "append only " + name,
		manifest: protectedManifest("append_only", "log.txt"),
		status:   status,
		message:  "protected path changed: mode=append_only path=log.txt status=" + diagnostic,
	}
}

func assertProtectedPathFailure(t *testing.T, test protectedPathCase) {
	t.Helper()

	t.Run(test.name, func(t *testing.T) {
		root := cleanRoot(t)
		installCommands(t, root, commandBehavior{GitStatus: test.status})
		writeFile(t, root, "quality/protected-paths.json", test.manifest)

		output, err := runScript(t, root, "--unit")

		requireCommandFailure(t, err)
		requireContains(t, output, test.message)
	})
}

func protectedManifest(mode string, pattern string) string {
	return `{
  "schema_version": 1,
  "path_root": "source_root",
  "protections": [{"mode":"` + mode + `","patterns":["` + pattern + `"],"reason":"fixture"}]
}`
}
