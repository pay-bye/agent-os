package checks

import "testing"

func TestConfigCheckUsesGoShimFromGoPath(t *testing.T) {
	root := releaseRoot(t, nil)
	installReleaseCommands(t, root)

	output, err := runReleaseScriptWithEnv(t, root, releaseCommandEnv(root), "--config")

	if err != nil {
		t.Fatalf("expected config check to pass, got %v\n%s", err, output)
	}
	requireContains(t, output, "config checked")
}

func TestSnapshotCheckSkipsPublishAndSign(t *testing.T) {
	root := releaseRoot(t, nil)
	installReleaseCommands(t, root)

	output, err := runReleaseScriptWithEnv(t, root, releaseCommandEnv(root), "--snapshot")

	if err != nil {
		t.Fatalf("expected snapshot check to pass, got %v\n%s", err, output)
	}
	requireContains(t, output, "snapshot checked")
}

func TestWorkflowChecksSnapshotThroughReleaseScript(t *testing.T) {
	workflow := readYAML(t, workflowPath(t))
	check := mapping(t, mapping(t, workflow, "jobs"), "check")
	snapshot := stepNamed(t, check, "Snapshot release")

	requireScalar(t, snapshot, "run", "./scripts/release-check.sh --snapshot")
	if _, ok := snapshot["uses"]; ok {
		t.Fatal("check snapshot must run through the release script")
	}
}

func TestWorkflowSubjectsComeFromReleaseSubjects(t *testing.T) {
	workflow := readYAML(t, workflowPath(t))
	publish := mapping(t, mapping(t, workflow, "jobs"), "publish")
	subjects := stepNamed(t, publish, "Generate subjects")

	requireScalar(t, subjects, "run", "./scripts/release-check.sh --subjects >> \"$GITHUB_OUTPUT\"")
}

func TestGuardAllowsCandidateAfterDestinations(t *testing.T) {
	output, err := runGuard(t, "v0.1.0-rc.1", []string{
		"GITHUB_REF_PROTECTED=true",
		"PUBLIC_RELEASE_DESTINATIONS_READY=true",
	})

	if err != nil {
		t.Fatalf("expected release candidate guard to pass, got %v\n%s", err, output)
	}
	requireContains(t, output, "publish eligible")
}

func TestGuardRejectsStableWithoutProof(t *testing.T) {
	output, err := runGuard(t, "v0.1.0", []string{
		"GITHUB_REF_PROTECTED=true",
		"PUBLIC_RELEASE_DESTINATIONS_READY=true",
	})

	requireCommandFailure(t, err)
	requireContains(t, output, "stable release requires parity proof evidence")
}

func TestGuardRejectsV1WithoutProductionEvidence(t *testing.T) {
	output, err := runGuard(t, "v1.0.0", []string{
		"GITHUB_REF_PROTECTED=true",
		"PUBLIC_RELEASE_DESTINATIONS_READY=true",
		"PARITY_PROOF_URI=https://evidence.example/parity",
	})

	requireCommandFailure(t, err)
	requireContains(t, output, "v1 release requires production and driver evidence")
}

func TestGuardRejectsNamespacedTag(t *testing.T) {
	output, err := runGuard(t, "agent-os/v0.1.0-rc.1", []string{
		"GITHUB_REF_PROTECTED=true",
		"PUBLIC_RELEASE_DESTINATIONS_READY=true",
	})

	requireCommandFailure(t, err)
	requireContains(t, output, "release tag must match vMAJOR.MINOR.PATCH")
}
