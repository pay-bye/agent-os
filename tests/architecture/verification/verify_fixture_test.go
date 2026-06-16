package checks

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type commandBehavior struct {
	WrongGoVersion      bool
	SlowFormat          bool
	GitStatus           string
	CoveragePercent     string
	ToolRunDelaySeconds int
}

func findRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if exists(filepath.Join(dir, "go.mod")) && exists(filepath.Join(dir, "scripts", "verify.sh")) {
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

func replaceScriptText(t *testing.T, root string, old string, new string) {
	t.Helper()

	for _, path := range copiedScriptPaths(t, root) {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		updated := strings.Replace(string(content), old, new, 1)
		if updated != string(content) {
			if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatalf("script text not found: %s", old)
}

func cleanRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeFile(t, root, "go.mod", "module github.com/pay-bye/agent-os\n\ngo 1.26.4\n")
	writeFile(t, root, "quality/protected-paths.json", `{"schema_version":1,"path_root":"source_root","protections":[]}`)
	writeFile(t, root, "quality/coverage-baseline.tsv", "package\tgate\tfloor_percent\treason\n")
	copyScript(t, root)
	return root
}

func copyScript(t *testing.T, root string) {
	t.Helper()

	copyVerificationFile(t, root, "scripts/verify.sh", 0o755)
	copyVerificationTree(t, root, "scripts/verify")
}

func copyVerificationTree(t *testing.T, root string, path string) {
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
		copyVerificationFile(t, root, relative, 0o644)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func copyVerificationFile(t *testing.T, root string, path string, mode os.FileMode) {
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

func copiedScriptPaths(t *testing.T, root string) []string {
	t.Helper()

	var paths []string
	scriptRoot := filepath.Join(root, "scripts")
	err := filepath.WalkDir(scriptRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && filepath.Ext(path) == ".sh" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return paths
}

func runScript(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()

	command := exec.Command(filepath.Join(root, "scripts", "verify.sh"), args...)
	command.Dir = root
	command.Env = commandEnvironment(root)
	output, err := command.CombinedOutput()
	return string(output), err
}

func runScriptWithExtraEnv(t *testing.T, root string, extraEnv []string, args ...string) (string, error) {
	t.Helper()

	command := exec.Command(filepath.Join(root, "scripts", "verify.sh"), args...)
	command.Dir = root
	command.Env = append(commandEnvironment(root), extraEnv...)
	output, err := command.CombinedOutput()
	return string(output), err
}

func commandEnvironment(root string) []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, item := range os.Environ() {
		if strings.HasPrefix(item, "PATH=") {
			continue
		}
		env = append(env, item)
	}
	path := filepath.Join(root, "bin") + string(os.PathListSeparator) + os.Getenv("PATH")
	return append(env, "PATH="+path)
}

func installCommands(t *testing.T, root string, behavior commandBehavior) {
	t.Helper()

	realGo, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	goroot := filepath.Join(root, "toolchain")
	gorootBin := filepath.Join(goroot, "bin")
	if err := os.MkdirAll(gorootBin, 0o755); err != nil {
		t.Fatal(err)
	}

	writeCommand(t, bin, "go", fakeBaseGo(root))
	writeCommand(t, bin, "go1.26.4", fakePinnedGo(goroot))
	writeCommand(t, bin, "git", fakeGit(behavior.GitStatus))
	writeCommand(t, bin, "gofmt", fakeFormat(behavior.SlowFormat))
	writeCommand(t, gorootBin, "go", fakeGo(realGo, behavior))
	writeCommand(t, gorootBin, "gofmt", fakeFormat(behavior.SlowFormat))
}

func writeCommand(t *testing.T, bin string, name string, content string) {
	t.Helper()

	path := filepath.Join(bin, name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func fakeBaseGo(root string) string {
	return "#!/usr/bin/env bash\n" +
		"set -euo pipefail\n" +
		"if [[ \"$1\" == \"env\" && \"$2\" == \"GOPATH\" ]]; then echo \"" + filepath.Join(root, "gopath") + "\"; exit 0; fi\n" +
		"exit 0\n"
}

func fakePinnedGo(goroot string) string {
	return "#!/usr/bin/env bash\n" +
		"set -euo pipefail\n" +
		"if [[ \"$1\" == \"env\" && \"$2\" == \"GOROOT\" ]]; then echo \"" + goroot + "\"; exit 0; fi\n" +
		"exit 0\n"
}

func fakeGo(realGo string, behavior commandBehavior) string {
	version := "go version go1.26.4 linux/amd64"
	if behavior.WrongGoVersion {
		version = "go version go1.24.0 linux/amd64"
	}
	coverage := behavior.CoveragePercent
	if coverage == "" {
		coverage = "100.0"
	}
	delay := strconv.Itoa(behavior.ToolRunDelaySeconds)

	return "#!/usr/bin/env bash\n" +
		"set -euo pipefail\n" +
		"delay_tool_run() { if (( " + delay + " > 0 )); then sleep " + delay + "; fi; }\n" +
		"require_scanner_pin() {\n" +
		"  case \"$1\" in\n" +
		"    \"honnef.co/go/tools/cmd/staticcheck@v0.7.0\") return 0 ;;\n" +
		"    \"github.com/securego/gosec/v2/cmd/gosec@v2.26.1\") return 0 ;;\n" +
		"    \"golang.org/x/vuln/cmd/govulncheck@v1.3.0\") return 0 ;;\n" +
		"    \"stepdown.dev/" + "go/cmd/stepdown@v0.1.1\") return 0 ;;\n" +
		"    *) echo \"unexpected scanner pin: $1\" >&2; exit 1 ;;\n" +
		"  esac\n" +
		"}\n" +
		"if [[ \"$1\" == \"version\" ]]; then echo \"" + version + "\"; exit 0; fi\n" +
		"if [[ \"$1\" == \"run\" && \"$2\" == /tmp/* ]]; then exec \"" + realGo + "\" \"$@\"; fi\n" +
		"if [[ \"$1\" == \"run\" ]]; then require_scanner_pin \"$2\"; delay_tool_run; exit 0; fi\n" +
		"if [[ \"$1\" == \"tool\" && \"$2\" == \"cover\" ]]; then echo \"total:\\t(statements)\\t" + coverage + "%\"; exit 0; fi\n" +
		"exit 0\n"
}

func fakeGit(status string) string {
	return "#!/usr/bin/env bash\n" +
		"set -euo pipefail\n" +
		"if [[ \"$1\" == \"status\" ]]; then cat <<'STATUS'\n" + status + "\nSTATUS\nexit 0; fi\n" +
		"exit 0\n"
}

func fakeFormat(slow bool) string {
	prefix := "#!/usr/bin/env bash\nset -euo pipefail\n"
	if slow {
		prefix += "sleep 6\n"
	}
	return prefix + "exit 0\n"
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
