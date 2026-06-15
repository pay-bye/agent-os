package checks

import (
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func findRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if exists(filepath.Join(dir, "go.mod")) && exists(filepath.Join(dir, "quality", "release")) {
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

func writeFile(t *testing.T, root string, path string, content string) {
	t.Helper()

	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readYAML(t *testing.T, path string) map[string]any {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(content, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func readText(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func workflowPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), ".github", "workflows", "release-agent-os.yml")
}

func repoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Clean(filepath.Join(findRoot(t), "..", "..", ".."))
}

func runGuard(t *testing.T, tag string, extraEnv []string) (string, error) {
	t.Helper()

	command := exec.Command(filepath.Join(findRoot(t), "scripts", "release-check.sh"), "--guard", tag)
	command.Dir = findRoot(t)
	command.Env = append(os.Environ(), extraEnv...)
	output, err := command.CombinedOutput()
	return string(output), err
}

func releaseRoot(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	copyReleaseScript(t, root)
	for path, content := range files {
		writeFile(t, root, path, content)
	}
	return root
}

func copyReleaseScript(t *testing.T, root string) {
	t.Helper()

	copyScriptFile(t, root, "scripts/release-check.sh", 0o755)
	copyScriptTree(t, root, "scripts/release")
}

func copyScriptTree(t *testing.T, root string, path string) {
	t.Helper()

	source := filepath.Join(findRoot(t), path)
	err := filepath.WalkDir(source, func(item string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(findRoot(t), item)
		if err != nil {
			return err
		}
		copyScriptFile(t, root, relative, 0o644)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func copyScriptFile(t *testing.T, root string, path string, mode os.FileMode) {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(findRoot(t), path))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, path, string(content))
	if err := os.Chmod(filepath.Join(root, path), mode); err != nil {
		t.Fatal(err)
	}
}

func runReleaseScript(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()

	command := exec.Command(filepath.Join(root, "scripts", "release-check.sh"), args...)
	command.Dir = root
	command.Env = os.Environ()
	output, err := command.CombinedOutput()
	return string(output), err
}

func runReleaseScriptWithEnv(t *testing.T, root string, env []string, args ...string) (string, error) {
	t.Helper()

	command := exec.Command(filepath.Join(root, "scripts", "release-check.sh"), args...)
	command.Dir = root
	command.Env = env
	output, err := command.CombinedOutput()
	return string(output), err
}

func releaseCommandEnv(root string) []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, item := range os.Environ() {
		if strings.HasPrefix(item, "PATH=") {
			continue
		}
		env = append(env, item)
	}
	path := strings.Join([]string{filepath.Join(root, "bin"), "/usr/bin", "/bin"}, string(os.PathListSeparator))
	return append(env, "PATH="+path)
}

func installReleaseCommands(t *testing.T, root string) {
	t.Helper()

	bin := filepath.Join(root, "bin")
	gopathBin := filepath.Join(root, "gopath", "bin")
	gorootBin := filepath.Join(root, "toolchain", "bin")
	for _, path := range []string{bin, gopathBin, gorootBin} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeExecutable(t, root, "bin/go", fakeBaseGo(root))
	writeExecutable(t, root, "gopath/bin/go1.26.4", fakePinnedGo(root))
	writeExecutable(t, root, "toolchain/bin/go", fakeToolchainGo())
	writeExecutable(t, root, "bin/curl", fakeReleaseRunner())
}

func writeExecutable(t *testing.T, root string, path string, content string) {
	t.Helper()

	writeFile(t, root, path, content)
	if err := os.Chmod(filepath.Join(root, path), 0o755); err != nil {
		t.Fatal(err)
	}
}

func fakeBaseGo(root string) string {
	return "#!/usr/bin/env bash\n" +
		"set -euo pipefail\n" +
		"if [[ \"$1\" == \"env\" && \"$2\" == \"GOPATH\" ]]; then echo \"" + filepath.Join(root, "gopath") + "\"; exit 0; fi\n" +
		"exit 1\n"
}

func fakePinnedGo(root string) string {
	return "#!/usr/bin/env bash\n" +
		"set -euo pipefail\n" +
		"if [[ \"$1\" == \"env\" && \"$2\" == \"GOROOT\" ]]; then echo \"" + filepath.Join(root, "toolchain") + "\"; exit 0; fi\n" +
		"exit 1\n"
}

func fakeToolchainGo() string {
	return "#!/usr/bin/env bash\n" +
		"set -euo pipefail\n" +
		"if [[ \"$1\" == \"version\" ]]; then echo \"go version go1.26.4 linux/amd64\"; exit 0; fi\n" +
		"exit 0\n"
}

func fakeReleaseRunner() string {
	return "#!/usr/bin/env bash\n" +
		"cat <<'SCRIPT'\n" +
		"#!/usr/bin/env bash\n" +
		"set -euo pipefail\n" +
		"if [[ \"$1\" == \"check\" && \"$2\" == \"--config\" && \"$3\" == \"quality/release/goreleaser.yaml\" ]]; then\n" +
		"  echo \"config checked\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"echo \"unexpected release args: $*\" >&2\n" +
		"exit 1\n" +
		"SCRIPT\n"
}

func decodedSubjects(t *testing.T, output string) string {
	t.Helper()

	encoded := strings.TrimSpace(strings.TrimPrefix(output, "base64="))
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("expected base64 output, got %q: %v", output, err)
	}
	return string(decoded)
}

func sequence(t *testing.T, item map[string]any, key string) []any {
	t.Helper()

	value, ok := item[key].([]any)
	if !ok {
		t.Fatalf("%s must be a sequence", key)
	}
	return value
}

func onlyItem(t *testing.T, items []any) map[string]any {
	t.Helper()

	if len(items) != 1 {
		t.Fatalf("expected one item, got %d", len(items))
	}
	value, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("sequence item must be a mapping")
	}
	return value
}

func mapping(t *testing.T, item map[string]any, key string) map[string]any {
	t.Helper()

	value, ok := item[key].(map[string]any)
	if !ok {
		t.Fatalf("%s must be a mapping", key)
	}
	return value
}

func requireMapping(t *testing.T, actual map[string]any, expected map[string]any) {
	t.Helper()

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("mapping = %v, want %v", actual, expected)
	}
}

func requireScalar(t *testing.T, item map[string]any, key string, want any) {
	t.Helper()

	if item[key] != want {
		t.Fatalf("%s = %v, want %v", key, item[key], want)
	}
}

func requireSequence(t *testing.T, item map[string]any, key string, want []string) {
	t.Helper()

	got := stringSequence(t, item, key)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %v, want %v", key, got, want)
	}
}

func stringSequence(t *testing.T, item map[string]any, key string) []string {
	t.Helper()

	values, ok := item[key].([]any)
	if !ok {
		t.Fatalf("%s must be a sequence", key)
	}
	items := make([]string, 0, len(values))
	for _, value := range values {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("%s item must be a string", key)
		}
		items = append(items, text)
	}
	return items
}

func firstString(t *testing.T, item map[string]any, key string) string {
	t.Helper()

	values := stringSequence(t, item, key)
	if len(values) == 0 {
		t.Fatalf("%s must not be empty", key)
	}
	return values[0]
}

func textValue(t *testing.T, item map[string]any, key string) string {
	t.Helper()

	value, ok := item[key].(string)
	if !ok {
		t.Fatalf("%s must be a string", key)
	}
	return value
}

func workflowUses(root any) []string {
	var items []string
	visitWorkflow(root, &items)
	return items
}

func stepNamed(t *testing.T, job map[string]any, name string) map[string]any {
	t.Helper()

	for _, step := range sequence(t, job, "steps") {
		item, ok := step.(map[string]any)
		if !ok {
			t.Fatal("workflow step must be a mapping")
		}
		if item["name"] == name {
			return item
		}
	}
	t.Fatalf("%s step not found", name)
	return nil
}

func visitWorkflow(value any, items *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		if use, ok := typed["uses"].(string); ok {
			*items = append(*items, use)
		}
		for _, next := range typed {
			visitWorkflow(next, items)
		}
	case []any:
		for _, next := range typed {
			visitWorkflow(next, items)
		}
	}
}

func pinnedAction(item string) bool {
	_, ref, ok := strings.Cut(item, "@")
	if !ok {
		return false
	}
	if regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(ref) {
		return true
	}
	return regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(ref)
}

func requireSliceContains[T comparable](t *testing.T, items []T, want T) {
	t.Helper()

	if !slices.Contains(items, want) {
		t.Fatalf("expected %v in %v", want, items)
	}
}

func requireCommandFailure(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected command failure")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected exit failure, got %v", err)
	}
}

func requireContains(t *testing.T, text string, want string) {
	t.Helper()

	if !strings.Contains(text, want) {
		t.Fatalf("expected output containing %q, got:\n%s", want, text)
	}
}
