package instructions

import (
	"reflect"
	"slices"
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestInstructionAuditsCarryPayloadFacts(t *testing.T) {
	move := moveAvailableAudit("x15", "x68", 2)
	drop := dropAudit([]workitem.ID{"x08", "x09"})
	route := routeAudit([]workitem.ID{"x08"}, []channel.EntryID{"x31"})

	requireAuditField(t, move, "source_channel_key", "x15")
	requireAuditField(t, move, "target_channel_key", "x68")
	requireAuditField(t, move, "limit", 2)
	requireAuditField(t, drop, "work_item_ids", []string{"x08", "x09"})
	requireAuditField(t, route, "entry_ids", []string{"x31"})
	requirePreconditions(t, route, "need_outstanding", "need_unrouted")
}

func TestInstructionAuditBuildersNamePreconditions(t *testing.T) {
	pause := pauseAudit("x17")
	lease := leaseAudit("x16", ExpiredLease)
	moveItem := moveItemAudit("x08", "x15", "x68")
	moveEntries := moveEntriesAudit("x15", "x68", []channel.EntryID{"x31"})

	requirePreconditions(t, pause, "node_installed", "node_has_alternate")
	requirePreconditions(t, lease, "lease_exists", "lease_expired")
	requirePreconditions(t, moveItem, "work_item_exists", "target_channel_routable")
	requirePreconditions(t, moveEntries, "entry_in_source", "no_current_lease")
}

func TestAuditPreconditionsAreCopied(t *testing.T) {
	preconditions := []string{"work_item_exists"}

	audit := newAudit(map[string]any{}, preconditions...)
	preconditions[0] = "mutated"

	requirePreconditions(t, audit, "work_item_exists")
}

func requireAuditField(t *testing.T, audit audit, name string, want any) {
	t.Helper()

	if !reflect.DeepEqual(audit.fields[name], want) {
		t.Fatalf("%s = %v, want %v", name, audit.fields[name], want)
	}
}

func requirePreconditions(t *testing.T, audit audit, expected ...string) {
	t.Helper()

	for _, item := range expected {
		if !slices.Contains(audit.preconditions, item) {
			t.Fatalf("preconditions = %v, missing %s", audit.preconditions, item)
		}
	}
}
