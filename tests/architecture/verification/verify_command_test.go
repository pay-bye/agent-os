package checks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeclaresPublicModulePath(t *testing.T) {
	const want = "github.com/pay-bye/agent-os"

	got := declaredModulePath(t, filepath.Join(findRoot(t), "go.mod"))

	if got != want {
		t.Fatalf("module path = %q, want %q", got, want)
	}
}

func TestUnknownFlagFailsClosed(t *testing.T) {
	root := cleanRoot(t)

	output, err := runScript(t, root, "--mystery")

	requireCommandFailure(t, err)
	requireContains(t, output, "unknown flag: --mystery")
}

func declaredModulePath(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1]
		}
	}
	t.Fatal("module path not found")
	return ""
}

func TestIntegrationRequiresControlPlaneEnvFile(t *testing.T) {
	root := cleanRoot(t)

	output, err := runScript(t, root, "--integration")

	requireCommandFailure(t, err)
	requireContains(t, output, "company/control-plane/control-plane.env.local")
	requireContains(t, output, "CONTROL_PLANE_TEST_DATABASE_URL")
	requireContains(t, output, "set -a; source company/control-plane/control-plane.env.local; set +a")
	requireContains(t, output, `export DATABASE_URL="${CONTROL_PLANE_TEST_DATABASE_URL:?}"`)
}

func TestIntegrationRequiresControlPlaneTestDatabaseURL(t *testing.T) {
	root := cleanRoot(t)
	writeFile(t, root, "company/control-plane/control-plane.env.local", "CONTROL_PLANE_PORT=8080\n")

	output, err := runScript(t, root, "--integration")

	requireCommandFailure(t, err)
	requireContains(t, output, "company/control-plane/control-plane.env.local")
	requireContains(t, output, "CONTROL_PLANE_TEST_DATABASE_URL")
}

func TestIntegrationRunsSuites(t *testing.T) {
	root := cleanRoot(t)
	writeControlPlaneEnv(t, root, "postgres://x47")
	installCommands(t, root, commandBehavior{})

	output, err := runScript(t, root, "--integration")

	if err != nil {
		t.Fatalf("expected integration gate to run suites, got %v\n%s", err, output)
	}
	requireContains(t, output, "integration tests")
}

func TestUnitRunsCleanSourceRoot(t *testing.T) {
	root := cleanRoot(t)
	installCommands(t, root, commandBehavior{})

	output, err := runScript(t, root, "--unit")

	if err != nil {
		t.Fatalf("expected clean source root to pass, got %v\n%s", err, output)
	}
	requireContains(t, output, "toolchain version verification")
	requireContains(t, output, "coverage floor check")
}

func TestUnitRejectsWrongGoVersion(t *testing.T) {
	root := cleanRoot(t)
	installCommands(t, root, commandBehavior{WrongGoVersion: true})

	output, err := runScript(t, root, "--unit")

	requireCommandFailure(t, err)
	requireContains(t, output, "go version mismatch")
}

func TestUnitRejectsScannerVersionDrift(t *testing.T) {
	for _, test := range scannerVersionCases() {
		assertScannerVersionFailure(t, test)
	}
}

func TestUnitRejectsStepdownVersionDrift(t *testing.T) {
	root := cleanRoot(t)
	replaceScriptText(
		t,
		root,
		"cmd/stepdown@v0.1.1",
		"cmd/stepdown@v0.1.2",
	)
	installCommands(t, root, commandBehavior{})

	output, err := runScript(t, root, "--unit")

	requireCommandFailure(t, err)
	requireContains(t, output, "stepdown version mismatch")
}

func TestUnitRejectsSlowStep(t *testing.T) {
	root := cleanRoot(t)
	installCommands(t, root, commandBehavior{SlowFormat: true})
	writeFile(t, root, "tests/sample_test.go", "package tests\n")

	output, err := runScript(t, root, "--unit")

	requireCommandFailure(t, err)
	requireContains(t, output, "step exceeded: gofmt clean check")
}

func TestUnitRejectsAggregateBudgetOverrun(t *testing.T) {
	root := cleanRoot(t)
	replaceScriptText(t, root, "readonly UNIT_LIMIT_SECONDS=300", "readonly UNIT_LIMIT_SECONDS=2")
	installCommands(t, root, commandBehavior{ToolRunDelaySeconds: 1})

	output, err := runScript(t, root, "--unit")

	requireCommandFailure(t, err)
	requireContains(t, output, "unit gate exceeded:")
}

type scannerVersionCase struct {
	name        string
	pinned      string
	replacement string
}

func scannerVersionCases() []scannerVersionCase {
	return []scannerVersionCase{
		{
			name:        "staticcheck",
			pinned:      "honnef.co/go/tools/cmd/staticcheck@v0.7.0",
			replacement: "honnef.co/go/tools/cmd/staticcheck@v0.7.1",
		},
		{
			name:        "gosec",
			pinned:      "github.com/securego/gosec/v2/cmd/gosec@v2.26.1",
			replacement: "github.com/securego/gosec/v2/cmd/gosec@v2.26.2",
		},
		{
			name:        "govulncheck",
			pinned:      "golang.org/x/vuln/cmd/govulncheck@v1.3.0",
			replacement: "golang.org/x/vuln/cmd/govulncheck@v1.3.1",
		},
	}
}

func assertScannerVersionFailure(t *testing.T, test scannerVersionCase) {
	t.Helper()

	t.Run(test.name, func(t *testing.T) {
		root := cleanRoot(t)
		replaceScriptText(t, root, test.pinned, test.replacement)
		installCommands(t, root, commandBehavior{})

		output, err := runScript(t, root, "--unit")

		requireCommandFailure(t, err)
		requireContains(t, output, "unexpected scanner pin")
	})
}

func writeControlPlaneEnv(t *testing.T, root string, databaseURL string) {
	t.Helper()

	writeFile(
		t,
		root,
		"company/control-plane/control-plane.env.local",
		"CONTROL_PLANE_TEST_DATABASE_URL='"+databaseURL+"'\n",
	)
}
