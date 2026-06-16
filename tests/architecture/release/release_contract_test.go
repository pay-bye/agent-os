package checks

import (
	"path/filepath"
	"testing"
)

func TestConfigBuildsAcceptedMatrix(t *testing.T) {
	config := readYAML(t, filepath.Join(findRoot(t), "quality", "release", "goreleaser.yaml"))
	build := onlyItem(t, sequence(t, config, "builds"))

	requireScalar(t, config, "pro", true)
	if _, ok := config["monorepo"]; ok {
		t.Fatal("public release config must use repository-root tags")
	}
	requireScalar(t, build, "main", "./cmd/substrate")
	requireScalar(t, build, "binary", "agent-os")
	requireSequence(t, build, "goos", []string{"linux", "darwin"})
	requireSequence(t, build, "goarch", []string{"amd64", "arm64"})
	requireSliceContains(t, stringSequence(t, build, "env"), "CGO_ENABLED=0")
}

func TestConfigPublishesAcceptedSurfaces(t *testing.T) {
	config := readYAML(t, filepath.Join(findRoot(t), "quality", "release", "goreleaser.yaml"))

	image := onlyItem(t, sequence(t, config, "dockers_v2"))
	requireContains(t, firstString(t, image, "images"), "ghcr.io/")
	requireSequence(t, image, "platforms", []string{"linux/amd64", "linux/arm64"})
	requireScalar(t, image, "sbom", true)

	cask := onlyItem(t, sequence(t, config, "homebrew_casks"))
	requireScalar(t, cask, "name", "agent-os")
	requireSliceContains(t, stringSequence(t, cask, "binaries"), "agent-os")
	requireScalar(t, cask, "directory", "Casks")
	if _, ok := cask["skip_upload"]; ok {
		t.Fatal("homebrew cask must publish to the configured tap")
	}

	signing := onlyItem(t, sequence(t, config, "signs"))
	requireScalar(t, signing, "if", "{{ not .IsSnapshot }}")
	requireScalar(t, signing, "cmd", "cosign")

	sbom := onlyItem(t, sequence(t, config, "sboms"))
	requireScalar(t, sbom, "cmd", "syft")
}

func TestInstallExamplesAreTyped(t *testing.T) {
	content := readText(t, filepath.Join(findRoot(t), "docs", "install.md"))

	for _, marker := range []string{"archive", "homebrew", "ghcr"} {
		requireContains(t, content, "<!-- install:"+marker+" -->")
	}
	requireContains(t, content, "cosign verify-blob")
	requireContains(t, content, "gh attestation verify")
	requireContains(t, content, "brew install --cask agent-os")
	requireContains(t, content, "ghcr.io/")
	requireContains(t, content, "--verifier-file")
}

func TestWorkflowGuardsPublish(t *testing.T) {
	workflow := readYAML(t, workflowPath(t))
	trigger := mapping(t, workflow, "on")
	push := mapping(t, trigger, "push")
	jobs := mapping(t, workflow, "jobs")
	publish := mapping(t, jobs, "publish")

	requireSequence(t, push, "tags", []string{"v*"})
	condition := textValue(t, publish, "if")
	requireContains(t, condition, "refs/tags/v")
	requireContains(t, condition, "github.ref_protected == true")
	requireContains(t, condition, "PUBLIC_RELEASE_DESTINATIONS_READY")
	requireMapping(t, mapping(t, publish, "permissions"), map[string]any{
		"contents":     "write",
		"packages":     "write",
		"id-token":     "write",
		"attestations": "write",
	})
	release := stepNamed(t, publish, "Release")
	requireScalar(t, mapping(t, release, "with"), "distribution", "goreleaser-pro")
	requireScalar(t, mapping(t, release, "env"), "GORELEASER_KEY", "${{ secrets.GORELEASER_KEY }}")
	requireScalar(t, mapping(t, release, "env"), "RELEASE_IMAGE_OWNER", "${{ vars.RELEASE_IMAGE_OWNER }}")
	requireScalar(t, mapping(t, release, "env"), "RELEASE_TAP_OWNER", "${{ vars.RELEASE_TAP_OWNER }}")
	requireScalar(t, mapping(t, release, "env"), "RELEASE_TAP_NAME", "${{ vars.RELEASE_TAP_NAME }}")
	requireScalar(t, mapping(t, release, "env"), "RELEASE_TAP_TOKEN", "${{ secrets.RELEASE_TAP_TOKEN }}")

	provenance := mapping(t, jobs, "provenance")
	requireScalar(t, mapping(t, provenance, "permissions"), "actions", "read")
}

func TestWorkflowInstallsPinnedGoShimBeforeChecks(t *testing.T) {
	workflow := readYAML(t, workflowPath(t))
	jobs := mapping(t, workflow, "jobs")

	for _, name := range []string{"check", "integration", "publish"} {
		job := mapping(t, jobs, name)
		shim := stepNamed(t, job, "Install pinned Go shim")

		requireContains(t, textValue(t, shim, "run"), "go install \"golang.org/dl/go${GO_VERSION}@latest\"")
		requireContains(t, textValue(t, shim, "run"), "\"$(go env GOPATH)/bin/go${GO_VERSION}\" download")
		requireContains(t, textValue(t, shim, "run"), "GITHUB_PATH")
		requireStepBefore(t, job, "Install pinned Go shim", firstCheckedStep(t, name))
	}
}

func TestWorkflowInstallsReleaseToolsBeforeSnapshot(t *testing.T) {
	workflow := readYAML(t, workflowPath(t))
	check := mapping(t, mapping(t, workflow, "jobs"), "check")
	tools := stepNamed(t, check, "Install release tools")

	requireContains(t, textValue(t, tools, "run"), "go install github.com/sigstore/cosign/v3/cmd/cosign@${COSIGN_VERSION}")
	requireContains(t, textValue(t, tools, "run"), "go install github.com/anchore/syft/cmd/syft@${SYFT_VERSION}")
	requireStepBefore(t, check, "Install release tools", "Check release config")
	requireStepBefore(t, check, "Install release tools", "Snapshot release")
}

func TestWorkflowUsesCurrentToolPins(t *testing.T) {
	workflow := readYAML(t, workflowPath(t))
	env := mapping(t, workflow, "env")

	requireScalar(t, env, "GO_VERSION", "1.26.4")
	requireScalar(t, env, "GORELEASER_VERSION", "v2.16.0")
	requireScalar(t, env, "COSIGN_VERSION", "v3.0.6")
	requireScalar(t, env, "SYFT_VERSION", "v1.44.0")
}

func TestWorkflowLocatesSourceRoot(t *testing.T) {
	workflow := readYAML(t, workflowPath(t))
	jobs := mapping(t, workflow, "jobs")

	for _, name := range []string{"check", "integration", "publish"} {
		job := mapping(t, jobs, name)
		source := stepNamed(t, job, "Locate source")

		requireScalar(t, source, "id", "source")
		requireContains(t, textValue(t, source, "run"), "root=.")
		requireContains(t, textValue(t, source, "run"), "company/software/agent-os")
		requireContains(t, textValue(t, source, "run"), "GITHUB_OUTPUT")
	}
}

func TestWorkflowRunsIntegrationWithPostgres(t *testing.T) {
	workflow := readYAML(t, workflowPath(t))
	integration := mapping(t, mapping(t, workflow, "jobs"), "integration")
	postgres := mapping(t, mapping(t, integration, "services"), "postgres")

	requireScalar(t, postgres, "image", "postgres:17.8-bookworm")
	requireMapping(t, mapping(t, postgres, "env"), map[string]any{
		"POSTGRES_DB":       "postgres",
		"POSTGRES_USER":     "postgres",
		"POSTGRES_PASSWORD": "postgres",
	})
	requireSequence(t, postgres, "ports", []string{"5432:5432"})
	requireContains(t, textValue(t, postgres, "options"), "pg_isready")
	verify := stepNamed(t, integration, "Verify integration")
	requireScalar(t, verify, "working-directory", "${{ steps.source.outputs.root }}")
	requireScalar(t, verify, "run", "./scripts/verify.sh --integration")
	requireScalar(
		t,
		mapping(t, verify, "env"),
		"CONTROL_PLANE_TEST_DATABASE_URL",
		"postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable",
	)
}

func TestWorkflowUsesPinnedActions(t *testing.T) {
	workflow := readYAML(t, workflowPath(t))
	uses := workflowUses(workflow)

	for _, expected := range []string{
		"actions/checkout@v6.0.2",
		"actions/setup-go@v6.4.0",
		"goreleaser/goreleaser-action@v7.2.2",
		"actions/attest@v4.1.0",
		"slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@v2.1.0",
	} {
		requireSliceContains(t, uses, expected)
	}
	for _, item := range uses {
		if !pinnedAction(item) {
			t.Fatalf("action is not pinned to a release tag or digest: %s", item)
		}
	}
}
