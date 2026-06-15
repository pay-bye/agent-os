package checks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnitRejectsCoverageBaselineFileProblems(t *testing.T) {
	for _, test := range coverageBaselineFileCases() {
		assertCoverageFailure(t, test)
	}
}

func TestUnitRejectsMissingCoverageFloor(t *testing.T) {
	assertCoverageFailure(t, coverageCase{
		name:     "missing floor",
		baseline: "package\tgate\tfloor_percent\treason\n",
		extra:    map[string]string{"internal/sample/file.go": "package sample\n"},
		message:  "coverage floor missing: github.com/pay-bye/agent-os/internal/sample gate=unit",
	})
}

func TestUnitRejectsBelowFloorCoverage(t *testing.T) {
	assertCoverageFailure(t, coverageCase{
		name:     "below floor",
		baseline: "package\tgate\tfloor_percent\treason\ngithub.com/pay-bye/agent-os/internal/sample\tunit\t80\tfixture\n",
		extra: map[string]string{
			"internal/sample/file.go":      "package sample\nfunc uncovered() int { return 1 }\n",
			"internal/sample/file_test.go": "package sample\nimport \"testing\"\nfunc TestPlaceholder(t *testing.T) {}\n",
		},
		coverage: "50.0",
		message:  "coverage below floor: github.com/pay-bye/agent-os/internal/sample gate=unit actual=0.0 floor=80",
	})
}

type coverageCase struct {
	name     string
	baseline string
	extra    map[string]string
	coverage string
	message  string
}

func coverageBaselineFileCases() []coverageCase {
	return []coverageCase{
		{
			name:     "missing",
			baseline: "",
			message:  "coverage baseline missing: quality/coverage-baseline.tsv",
		},
		{
			name:     "malformed",
			baseline: "package\tgate\tfloor_percent\treason\ngithub.com/pay-bye/agent-os/internal/sample\tunit\t101\tbad\n",
			message:  "coverage baseline invalid",
		},
		{
			name:     "unknown gate",
			baseline: "package\tgate\tfloor_percent\treason\ngithub.com/pay-bye/agent-os/internal/sample\treadiness\t90\tbad\n",
			message:  "coverage baseline invalid",
		},
		{
			name:     "duplicate",
			baseline: "package\tgate\tfloor_percent\treason\ngithub.com/pay-bye/agent-os/internal/sample\tunit\t80\tone\ngithub.com/pay-bye/agent-os/internal/sample\tunit\t80\ttwo\n",
			message:  "coverage baseline invalid",
		},
	}
}

func assertCoverageFailure(t *testing.T, test coverageCase) {
	t.Helper()

	t.Run(test.name, func(t *testing.T) {
		root := cleanRoot(t)
		installCommands(t, root, commandBehavior{CoveragePercent: test.coverage})
		writeCoverageBaseline(t, root, test.baseline)
		writeExtraFiles(t, root, test.extra)

		output, err := runScript(t, root, "--unit")

		requireCommandFailure(t, err)
		requireContains(t, output, test.message)
	})
}

func writeCoverageBaseline(t *testing.T, root string, baseline string) {
	t.Helper()

	if baseline == "" {
		if err := os.Remove(filepath.Join(root, "quality", "coverage-baseline.tsv")); err != nil {
			t.Fatal(err)
		}
		return
	}
	writeFile(t, root, "quality/coverage-baseline.tsv", baseline)
}

func writeExtraFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()

	for path, content := range files {
		writeFile(t, root, path, content)
	}
}
