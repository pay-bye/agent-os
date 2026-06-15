package release_test

import (
	"strings"
	"testing"
)

func TestPayloadReferencesPublishedReleaseSurfaces(t *testing.T) {
	payload := validPayload()

	requireSourceRepository(t, payload, "example/agent-os")
	requireHomebrewTap(t, payload, "example/homebrew-tap")
	requireReleaseURL(t, payload, "https://github.com/example/agent-os/releases/tag/v0.1.0-rc.1")
	requireImageOwners(t, payload, "ghcr.io/example/agent-os")
	requireReferenceOwners(t, payload, "https://github.com/example/agent-os/releases/download/")
}

func requireSourceRepository(t *testing.T, payload map[string]any, want string) {
	t.Helper()

	source := payload["source"].(map[string]any)
	if source["repository"] != want {
		t.Fatalf("source repository = %v, want %s", source["repository"], want)
	}
}

func requireHomebrewTap(t *testing.T, payload map[string]any, want string) {
	t.Helper()

	homebrew := payload["homebrew"].(map[string]any)
	if homebrew["tap_repository"] != want {
		t.Fatalf("tap repository = %v, want %s", homebrew["tap_repository"], want)
	}
}

func requireReleaseURL(t *testing.T, payload map[string]any, want string) {
	t.Helper()

	notes := payload["notes"].(map[string]any)
	if notes["release_url"] != want {
		t.Fatalf("release URL = %v, want %s", notes["release_url"], want)
	}
}

func requireImageOwners(t *testing.T, payload map[string]any, want string) {
	t.Helper()

	for _, item := range payload["images"].([]any) {
		image := item.(map[string]any)
		if image["name"] != want {
			t.Fatalf("image name = %v, want %s", image["name"], want)
		}
		signature := image["signature"].(string)
		if !strings.HasPrefix(signature, want+"@") {
			t.Fatalf("image signature = %s, want %s digest reference", signature, want)
		}
	}
}

func requireReferenceOwners(t *testing.T, payload map[string]any, prefix string) {
	t.Helper()

	for _, section := range []string{"sboms", "signatures", "provenance", "attestations"} {
		reference := firstReference(t, payload, section)
		uri := reference["uri"].(string)
		if !strings.HasPrefix(uri, prefix) {
			t.Fatalf("%s reference URI = %s, want prefix %s", section, uri, prefix)
		}
	}
}
