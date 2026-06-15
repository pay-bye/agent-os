package declaration

import (
	"testing"

	"github.com/pay-bye/agent-os/internal/storage/postgres"
)

func TestBuildDeltaReportsInstallableAdditionsAndConflicts(t *testing.T) {
	document := mustParse(t, validDocument())
	empty := postgres.Catalog{}
	changed := document.Vocabulary()
	changed.Items[0].Description = "Other"

	additions := BuildDelta(empty, document.Vocabulary())
	conflicts := BuildDelta(changed, document.Vocabulary())

	if !additions.Installable || len(additions.Additions) == 0 {
		t.Fatalf("additions = %+v", additions)
	}
	if conflicts.Installable || len(conflicts.Conflicts) == 0 {
		t.Fatalf("conflicts = %+v", conflicts)
	}
}

func TestBuildDeltaReportsRemovalsAndClearances(t *testing.T) {
	document := mustParse(t, validDocument())
	current := document.Vocabulary()
	current.Routes = append(current.Routes, postgres.RouteRecord{Need: "x12", Node: "x17", Order: 2})
	current.Nodes = append(current.Nodes, postgres.NodeRecord{Key: "x18", Description: "x24", Channel: "x18", ChannelLabel: "x24", Accepts: []string{"x12"}})
	current.RoutingExclusions = []postgres.RoutingExclusionRecord{{Node: "x17"}, {Node: "x18"}}

	delta := BuildDelta(current, document.Vocabulary())

	if !delta.Installable {
		t.Fatalf("delta = %+v, want installable", delta)
	}
	requireRef(t, delta.Removals, "route", "x12/002")
	requireRef(t, delta.Removals, "node", "x18")
	requireRef(t, delta.Clearances, "routing_exclusion", "x17")
	if hasRef(delta.Clearances, "routing_exclusion", "x18") {
		t.Fatalf("clearances = %+v, did not expect removed node clearance", delta.Clearances)
	}
}

func requireRef(t *testing.T, refs []RecordRef, kind string, key string) {
	t.Helper()

	if !hasRef(refs, kind, key) {
		t.Fatalf("refs = %+v, want %s %s", refs, kind, key)
	}
}

func hasRef(refs []RecordRef, kind string, key string) bool {
	for _, ref := range refs {
		if ref.Kind == kind && ref.Key == key {
			return true
		}
	}
	return false
}
