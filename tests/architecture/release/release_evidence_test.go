package checks

import (
	"strings"
	"testing"
)

func TestSubjectOutputCoversArchivesAndImages(t *testing.T) {
	root := releaseRoot(t, map[string]string{
		"dist/release/checksums.txt": strings.Repeat("a", 64) + "  agent-os_v0.1.0_linux_amd64.tar.gz\n",
		"dist/release/digests.txt":   "sha256:" + strings.Repeat("b", 64) + "  ghcr.io/example/agent-os:v0.1.0\n",
	})

	output, err := runReleaseScript(t, root, "--subjects")

	if err != nil {
		t.Fatalf("expected subject generation to pass, got %v\n%s", err, output)
	}
	decoded := decodedSubjects(t, output)
	requireContains(t, decoded, strings.Repeat("a", 64)+"  agent-os_v0.1.0_linux_amd64.tar.gz")
	requireContains(t, decoded, strings.Repeat("b", 64)+"  ghcr.io/example/agent-os:v0.1.0")
	if strings.Contains(decoded, "sha256:") {
		t.Fatalf("image subjects must trim sha256 prefix, got:\n%s", decoded)
	}
}
