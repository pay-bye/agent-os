package http

import (
	"crypto/sha256"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestCompatibilityReturnsContractMetadataWithoutCommandExecution(t *testing.T) {
	commands := &recordingCommands{}
	request := httptest.NewRequest("GET", "/compatibility", nil)

	response := serveRequest(t, commands, request)

	requireCode(t, response, 200)
	requireJSONContent(t, response)
	requireCalls(t, commands.calls)
	requireBody(t, response, expectedCompatibility(t))
}

func expectedCompatibility(t *testing.T) map[string]any {
	t.Helper()

	return map[string]any{
		"contract_version":  "v1",
		"schema_set_digest": schemaSetDigest(t),
		"features": []any{
			"lease_claim",
			"lease_extend",
			"lease_ack",
			"lease_nack",
			"lease_capability",
			"declared_needs",
			"failure_payload",
		},
		"routes": []any{
			map[string]any{"method": "POST", "path": "/submit"},
			map[string]any{"method": "POST", "path": "/claim"},
			map[string]any{"method": "POST", "path": "/ack"},
			map[string]any{"method": "POST", "path": "/nack"},
			map[string]any{"method": "POST", "path": "/extend"},
			map[string]any{"method": "POST", "path": "/heartbeat"},
			map[string]any{"method": "POST", "path": "/operations/instructions/pause"},
			map[string]any{"method": "POST", "path": "/operations/instructions/release-expired-lease"},
			map[string]any{"method": "POST", "path": "/operations/instructions/force-release-lease"},
			map[string]any{"method": "POST", "path": "/operations/instructions/move-item"},
			map[string]any{"method": "POST", "path": "/operations/instructions/move-entries"},
			map[string]any{"method": "POST", "path": "/operations/instructions/move-available"},
			map[string]any{"method": "POST", "path": "/operations/instructions/drop"},
			map[string]any{"method": "POST", "path": "/operations/instructions/route-outstanding"},
			map[string]any{"method": "GET", "path": "/health"},
			map[string]any{"method": "GET", "path": "/readyz"},
			map[string]any{"method": "GET", "path": "/metrics"},
			map[string]any{"method": "GET", "path": "/operations"},
			map[string]any{"method": "GET", "path": "/operations/channels"},
			map[string]any{"method": "GET", "path": "/operations/channels/{channel_key}/items"},
			map[string]any{"method": "GET", "path": "/operations/items/{work_item_id}"},
			map[string]any{"method": "GET", "path": "/operations/items/{work_item_id}/journal"},
			map[string]any{"method": "GET", "path": "/operations/nodes"},
			map[string]any{"method": "GET", "path": "/compatibility"},
		},
	}
}

func schemaSetDigest(t *testing.T) string {
	t.Helper()

	root := contractRoot(t)
	paths := contractFiles(t, root)
	hash := sha256.New()
	for _, path := range paths {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatal(err)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		hash.Write([]byte(filepath.ToSlash(relative)))
		hash.Write([]byte{0})
		hash.Write(content)
		hash.Write([]byte{0})
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil))
}

func contractRoot(t *testing.T) string {
	t.Helper()

	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		root := filepath.Join(current, "contracts", "invocation")
		if pathExists(root) {
			return root
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatal("contract root not found")
		}
		current = parent
	}
}

func contractFiles(t *testing.T, root string) []string {
	t.Helper()

	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(files)
	return files
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
