package checks

import (
	"path/filepath"
	"testing"
)

func TestReleaseWorkflowPathAcceptsPublicRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".github/workflows/release-agent-os.yml", "name: release agent-os\n")

	path, ok := releaseWorkflowPath(root)

	if !ok {
		t.Fatal("expected release workflow path")
	}
	want := filepath.Join(root, ".github", "workflows", "release-agent-os.yml")
	if path != want {
		t.Fatalf("unexpected release workflow path: got %s want %s", path, want)
	}
}
