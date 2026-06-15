package payloads

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEventKindsLiveInDedicatedFiles(t *testing.T) {
	files := sourceFiles(t)
	for _, event := range dedicatedEvents() {
		body, ok := files[event.file]
		if !ok {
			t.Fatalf("%s missing for %s", event.file, event.kind)
		}
		if !strings.Contains(body, event.kind) {
			t.Fatalf("%s does not contain %s", event.file, event.kind)
		}
		if !strings.Contains(body, event.codec) {
			t.Fatalf("%s does not contain %s", event.file, event.codec)
		}
		for name, candidate := range files {
			if name != event.file && strings.Contains(candidate, event.kind) {
				t.Fatalf("%s also contains %s", name, event.kind)
			}
		}
	}
}

type dedicatedEvent struct {
	file  string
	kind  string
	codec string
}

func dedicatedEvents() []dedicatedEvent {
	return []dedicatedEvent{
		{file: "item_submitted.go", kind: "ItemSubmittedKind", codec: "func ItemSubmitted"},
		{file: "need_declared.go", kind: "NeedDeclaredKind", codec: "func NeedDeclared"},
		{file: "item_routed.go", kind: "ItemRoutedKind", codec: "func ItemRouted"},
		{file: "need_acked.go", kind: "NeedAckedKind", codec: "func NeedAcked"},
		{file: "need_nacked.go", kind: "NeedNackedKind", codec: "func NeedNacked"},
		{file: "exclusion_set.go", kind: "ExclusionSetKind", codec: "func ExclusionSet"},
		{file: "exclusion_clear.go", kind: "ExclusionClearKind", codec: "func ExclusionClear"},
		{file: "instruction_applied.go", kind: "InstructionAppliedKind", codec: "func InstructionApplied"},
		{file: "instruction_rejected.go", kind: "InstructionRejectedKind", codec: "func InstructionRejected"},
		{file: "work_item_dropped.go", kind: "WorkItemDroppedKind", codec: "func WorkItemDropped"},
	}
}

func sourceFiles(t *testing.T) map[string]string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("current file not available")
	}
	entries, err := os.ReadDir(filepath.Dir(currentFile))
	if err != nil {
		t.Fatal(err)
	}
	files := make(map[string]string)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(filepath.Dir(currentFile), name))
		if err != nil {
			t.Fatal(err)
		}
		files[name] = string(body)
	}
	return files
}
