//go:build integration

package instructions_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/substrate"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestInstructionJournalPayloadsContainAuditFacts(t *testing.T) {
	tests := []instructionPayloadCase{
		pausePayloadCase(),
		releaseExpiredPayloadCase(),
		forceReleasePayloadCase(),
		moveItemPayloadCase(),
		moveEntriesPayloadCase(),
		moveAvailablePayloadCase(),
		dropPayloadCase(),
		routeOutstandingPayloadCase(),
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			db, schema := fixture.MigratedSchema(t, ctx)
			test.setup(t, ctx, db, schema)

			if _, err := test.run(ctx, db, schema); err != nil {
				t.Fatal(err)
			}

			body := journalPayload(t, ctx, db, schema, test.event)
			requirePayloadFields(t, body, test.fields)
			requirePreconditions(t, body, test.preconditions...)
			requireNoForbiddenInstructionFields(t, body)
		})
	}
}

func TestInstructionPauseReplaysDuplicateWithoutRemutation(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.InsertCapability(t, ctx, db, schema, "x18", "x12")

	first := fixture.CommandsFor(db, schema, "x80")
	firstResult, err := first.PauseInstruction(ctx, kernel.PauseInstructionInput{
		ID:   "x70",
		Node: registry.NodeKey("x17"),
	})
	if err != nil {
		t.Fatal(err)
	}
	second := fixture.CommandsFor(db, schema, "x81")
	secondResult, err := second.PauseInstruction(ctx, kernel.PauseInstructionInput{
		ID:   "x70",
		Node: registry.NodeKey("x17"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if !firstResult.SameReplay(secondResult) {
		t.Fatalf("replay = %+v, want %+v", secondResult, firstResult)
	}
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM routing_exclusions WHERE node_key = 'x17'`, 1)
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM instruction_records WHERE instruction_id = 'x70'`, 1)
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM journal_events WHERE event_kind_key = 'x47'`, 1)
}

func TestInstructionReplayReadsOutcomeFromJournalPayload(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.InsertCapability(t, ctx, db, schema, "x18", "x12")
	first := fixture.CommandsFor(db, schema, "x80")
	firstResult, err := first.PauseInstruction(ctx, kernel.PauseInstructionInput{
		ID:   "x70",
		Node: registry.NodeKey("x17"),
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture.RequireScalar(t, ctx, db, schema, `
SELECT count(*)
FROM information_schema.columns
WHERE table_schema = current_schema()
  AND table_name = 'instruction_records'
  AND column_name IN ('result', 'affected_ids', 'failed_precondition')`, 0)

	second := fixture.CommandsFor(db, schema, "x81")
	secondResult, err := second.PauseInstruction(ctx, kernel.PauseInstructionInput{
		ID:   "x70",
		Node: registry.NodeKey("x17"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if !firstResult.SameReplay(secondResult) {
		t.Fatalf("replay = %+v, want %+v", secondResult, firstResult)
	}
}

func TestInstructionKeyConflictDoesNotAppendJournal(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.InsertCapability(t, ctx, db, schema, "x18", "x12")
	commands := fixture.CommandsFor(db, schema, "x80", "x81")
	if _, err := commands.PauseInstruction(ctx, kernel.PauseInstructionInput{
		ID:   "x70",
		Node: registry.NodeKey("x17"),
	}); err != nil {
		t.Fatal(err)
	}

	_, err := commands.ReleaseExpiredLeaseInstruction(ctx, kernel.LeaseInstructionInput{
		ID:    "x70",
		Lease: channel.LeaseID("x16"),
	})

	if !errors.Is(err, kernel.ErrInstructionConflict) {
		t.Fatalf("error = %v, want instruction conflict", err)
	}
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM journal_events WHERE event_kind_key = 'x48'`, 0)
}

func TestInstructionReleaseCommandsKeepLeaseWindowsDistinct(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.SubmitRoutedItem(t, ctx, db, schema)
	fixture.ClaimAlpha(t, ctx, db, schema)

	early := fixture.CommandsAt(db, schema, fixture.Instant(1), "x80")
	result, err := early.ReleaseExpiredLeaseInstruction(ctx, kernel.LeaseInstructionInput{
		ID:    "x70",
		Lease: channel.LeaseID("x16"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Result != kernel.InstructionPreconditionFailed || result.FailedPrecondition != "lease_expired" {
		t.Fatalf("result = %+v, want lease_expired precondition", result)
	}
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM leases WHERE id = 'x16'`, 1)

	late := fixture.CommandsAt(db, schema, fixture.Instant(20), "x81")
	result, err = late.ReleaseExpiredLeaseInstruction(ctx, kernel.LeaseInstructionInput{
		ID:    "x71",
		Lease: channel.LeaseID("x16"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Result != kernel.InstructionApplied {
		t.Fatalf("result = %+v, want applied", result)
	}
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM leases WHERE id = 'x16'`, 0)

	result, err = late.ForceReleaseLeaseInstruction(ctx, kernel.LeaseInstructionInput{
		ID:    "x72",
		Lease: channel.LeaseID("x16"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Result != kernel.InstructionPreconditionFailed || result.FailedPrecondition != "lease_exists" {
		t.Fatalf("result = %+v, want lease_exists precondition", result)
	}
}

func TestInstructionMoveAvailableMovesOldestEntriesAndRecordsSelection(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	insertQueuedItem(t, ctx, db, schema, "x31", "x09", fixture.Instant(-2))
	insertQueuedItem(t, ctx, db, schema, "x32", "x10", fixture.Instant(-1))
	insertQueuedItem(t, ctx, db, schema, "x33", "x11", fixture.Instant(0))
	commands := fixture.CommandsFor(db, schema, "x80")

	result, err := commands.MoveAvailableInstruction(ctx, kernel.MoveAvailableInstructionInput{
		ID:     "x70",
		Source: registry.ChannelKey("x15"),
		Target: registry.ChannelKey("x68"),
		Limit:  2,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Result != kernel.InstructionApplied || !fixture.EqualStrings(result.AffectedIDs, []string{"x31", "x32"}) {
		t.Fatalf("result = %+v, want first two entries", result)
	}
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM channel_entries WHERE channel_key = 'x68'`, 2)
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM channel_entries WHERE channel_key = 'x15'`, 1)
}

func TestInstructionDropKeepsWorkItemAndJournalHistory(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	fixture.SubmitRoutedItem(t, ctx, db, schema)
	commands := fixture.CommandsFor(db, schema, "x80", "x81")

	result, err := commands.DropInstruction(ctx, kernel.ItemsInstructionInput{
		ID:        "x70",
		WorkItems: []workitem.ID{workitem.ID("x08")},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Result != kernel.InstructionApplied {
		t.Fatalf("result = %+v, want applied", result)
	}
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM work_items WHERE id = 'x08'`, 1)
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM channel_entries WHERE work_item_id = 'x08'`, 0)
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM journal_events WHERE event_kind_key = 'x49'`, 1)

	result, err = commands.RouteOutstandingInstruction(ctx, kernel.ItemsInstructionInput{
		ID:        "x71",
		WorkItems: []workitem.ID{workitem.ID("x08")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Result != kernel.InstructionPreconditionFailed || result.FailedPrecondition != "work_item_not_terminal" {
		t.Fatalf("result = %+v, want terminal precondition", result)
	}
}

func TestInstructionRouteOutstandingUsesAddressedRouterWithoutFallback(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	insertSubmittedItem(t, ctx, db, schema, "x08", "x18")
	commands := fixture.CommandsFor(db, schema, "x80", "x81")

	result, err := commands.RouteOutstandingInstruction(ctx, kernel.ItemsInstructionInput{
		ID:        "x70",
		WorkItems: []workitem.ID{workitem.ID("x08")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Result != kernel.InstructionPreconditionFailed || result.FailedPrecondition != "target_channel_routable" {
		t.Fatalf("result = %+v, want addressed failure", result)
	}
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM channel_entries WHERE work_item_id = 'x08'`, 0)
}

func TestInstructionRouteOutstandingRecordsInstructionAndRoutingFacts(t *testing.T) {
	ctx := context.Background()
	db, schema := fixture.MigratedSchema(t, ctx)
	fixture.InsertVocabulary(t, ctx, db, schema)
	insertSubmittedItem(t, ctx, db, schema, "x08", "")
	commands := fixture.CommandsFor(db, schema, "x80", "x81", "x31")

	result, err := commands.RouteOutstandingInstruction(ctx, kernel.ItemsInstructionInput{
		ID:        "x70",
		WorkItems: []workitem.ID{workitem.ID("x08")},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Result != kernel.InstructionApplied || !fixture.EqualStrings(result.EventIDs, []string{"x80", "x81"}) {
		t.Fatalf("result = %+v, want instruction and routing events", result)
	}
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM journal_events WHERE event_kind_key = 'x47'`, 1)
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM journal_events WHERE event_kind_key = 'x42'`, 1)
	fixture.RequireScalar(t, ctx, db, schema, `SELECT count(*) FROM channel_entries WHERE work_item_id = 'x08'`, 1)
}

type instructionPayloadCase struct {
	name          string
	event         string
	fields        map[string]any
	preconditions []string
	setup         func(*testing.T, context.Context, *sql.DB, string)
	run           func(context.Context, *sql.DB, string) (kernel.InstructionResult, error)
}

func pausePayloadCase() instructionPayloadCase {
	return instructionPayloadCase{
		name:          "pause",
		event:         "x80",
		fields:        map[string]any{"node_key": "x17"},
		preconditions: []string{"node_installed", "node_has_alternate"},
		setup: func(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
			fixture.InsertVocabulary(t, ctx, db, schema)
			fixture.InsertCapability(t, ctx, db, schema, "x18", "x12")
		},
		run: func(ctx context.Context, db *sql.DB, schema string) (kernel.InstructionResult, error) {
			return fixture.CommandsFor(db, schema, "x80").PauseInstruction(ctx, kernel.PauseInstructionInput{
				ID:   "x70",
				Node: registry.NodeKey("x17"),
			})
		},
	}
}

func releaseExpiredPayloadCase() instructionPayloadCase {
	return leasePayloadCase("release-expired", expiredCommand)
}

func forceReleasePayloadCase() instructionPayloadCase {
	return leasePayloadCase("force-release", forceCommand)
}

type leaseCommand string

const (
	expiredCommand leaseCommand = "expired"
	forceCommand   leaseCommand = "force"
)

func leasePayloadCase(name string, command leaseCommand) instructionPayloadCase {
	precondition := "lease_expired"
	at := fixture.Instant(20)
	if command == forceCommand {
		precondition = "lease_unexpired"
		at = fixture.Instant(1)
	}
	return instructionPayloadCase{
		name:          name,
		event:         "x80",
		fields:        map[string]any{"lease_id": "x16"},
		preconditions: []string{"lease_exists", precondition},
		setup: func(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
			fixture.InsertVocabulary(t, ctx, db, schema)
			fixture.SubmitRoutedItem(t, ctx, db, schema)
			fixture.ClaimAlpha(t, ctx, db, schema)
		},
		run: func(ctx context.Context, db *sql.DB, schema string) (kernel.InstructionResult, error) {
			input := kernel.LeaseInstructionInput{ID: "x70", Lease: channel.LeaseID("x16")}
			commands := fixture.CommandsAt(db, schema, at, "x80")
			if command == forceCommand {
				return commands.ForceReleaseLeaseInstruction(ctx, input)
			}
			return commands.ReleaseExpiredLeaseInstruction(ctx, input)
		},
	}
}

func moveItemPayloadCase() instructionPayloadCase {
	return instructionPayloadCase{
		name:  "move-item",
		event: "x80",
		fields: map[string]any{
			"work_item_id":       "x08",
			"source_channel_key": "x15",
			"target_channel_key": "x68",
		},
		preconditions: []string{
			"work_item_exists",
			"work_item_not_terminal",
			"channel_exists",
			"target_channel_routable",
			"entry_exists",
			"entry_in_source",
			"no_current_lease",
		},
		setup: func(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
			fixture.InsertVocabulary(t, ctx, db, schema)
			fixture.SubmitRoutedItem(t, ctx, db, schema)
		},
		run: func(ctx context.Context, db *sql.DB, schema string) (kernel.InstructionResult, error) {
			return fixture.CommandsFor(db, schema, "x80").MoveItemInstruction(ctx, kernel.MoveItemInstructionInput{
				ID:       "x70",
				WorkItem: workitem.ID("x08"),
				Source:   registry.ChannelKey("x15"),
				Target:   registry.ChannelKey("x68"),
			})
		},
	}
}

func moveEntriesPayloadCase() instructionPayloadCase {
	return instructionPayloadCase{
		name:  "move-entries",
		event: "x80",
		fields: map[string]any{
			"source_channel_key": "x15",
			"target_channel_key": "x68",
			"entry_ids":          []string{"x31"},
		},
		preconditions: []string{
			"channel_exists",
			"target_channel_routable",
			"entry_exists",
			"entry_in_source",
			"no_current_lease",
		},
		setup: func(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
			fixture.InsertVocabulary(t, ctx, db, schema)
			insertQueuedItem(t, ctx, db, schema, "x31", "x09", fixture.Instant(0))
		},
		run: func(ctx context.Context, db *sql.DB, schema string) (kernel.InstructionResult, error) {
			return fixture.CommandsFor(db, schema, "x80").MoveEntriesInstruction(ctx, kernel.MoveEntriesInstructionInput{
				ID:      "x70",
				Source:  registry.ChannelKey("x15"),
				Target:  registry.ChannelKey("x68"),
				Entries: []channel.EntryID{channel.EntryID("x31")},
			})
		},
	}
}

func moveAvailablePayloadCase() instructionPayloadCase {
	return instructionPayloadCase{
		name:  "move-available",
		event: "x80",
		fields: map[string]any{
			"source_channel_key": "x15",
			"target_channel_key": "x68",
			"limit":              float64(2),
		},
		preconditions: []string{"channel_exists", "target_channel_routable", "limit"},
		setup: func(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
			fixture.InsertVocabulary(t, ctx, db, schema)
			insertQueuedItem(t, ctx, db, schema, "x31", "x09", fixture.Instant(-2))
			insertQueuedItem(t, ctx, db, schema, "x32", "x10", fixture.Instant(-1))
		},
		run: func(ctx context.Context, db *sql.DB, schema string) (kernel.InstructionResult, error) {
			return fixture.CommandsFor(db, schema, "x80").MoveAvailableInstruction(ctx, kernel.MoveAvailableInstructionInput{
				ID:     "x70",
				Source: registry.ChannelKey("x15"),
				Target: registry.ChannelKey("x68"),
				Limit:  2,
			})
		},
	}
}

func dropPayloadCase() instructionPayloadCase {
	return instructionPayloadCase{
		name:          "drop",
		event:         "x80",
		fields:        map[string]any{"work_item_ids": []string{"x08"}},
		preconditions: []string{"work_item_exists", "work_item_not_terminal", "no_current_lease"},
		setup: func(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
			fixture.InsertVocabulary(t, ctx, db, schema)
			fixture.SubmitRoutedItem(t, ctx, db, schema)
		},
		run: func(ctx context.Context, db *sql.DB, schema string) (kernel.InstructionResult, error) {
			return fixture.CommandsFor(db, schema, "x80").DropInstruction(ctx, kernel.ItemsInstructionInput{
				ID:        "x70",
				WorkItems: []workitem.ID{workitem.ID("x08")},
			})
		},
	}
}

func routeOutstandingPayloadCase() instructionPayloadCase {
	return instructionPayloadCase{
		name:  "route-outstanding",
		event: "x80",
		fields: map[string]any{
			"work_item_ids": []string{"x08"},
			"entry_ids":     []string{"x31"},
		},
		preconditions: []string{
			"work_item_exists",
			"work_item_not_terminal",
			"no_current_lease",
			"need_outstanding",
			"need_unrouted",
			"target_channel_routable",
		},
		setup: func(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
			fixture.InsertVocabulary(t, ctx, db, schema)
			insertSubmittedItem(t, ctx, db, schema, "x08", "")
		},
		run: func(ctx context.Context, db *sql.DB, schema string) (kernel.InstructionResult, error) {
			return fixture.CommandsFor(db, schema, "x80", "x81", "x31").RouteOutstandingInstruction(
				ctx,
				kernel.ItemsInstructionInput{
					ID:        "x70",
					WorkItems: []workitem.ID{workitem.ID("x08")},
				},
			)
		},
	}
}

func insertQueuedItem(
	t *testing.T,
	ctx context.Context,
	db queryDB,
	schema string,
	entry string,
	item string,
	availableAt time.Time,
) {
	t.Helper()

	insertSubmittedItem(t, ctx, db, schema, item, "")
	_, err := db.ExecContext(ctx, `
INSERT INTO `+schema+`.channel_entries (id, channel_key, work_item_id, enqueued_at, available_at)
VALUES ($1, 'x15', $2, $3, $4)`, entry, item, fixture.Instant(-3), availableAt)
	if err != nil {
		t.Fatal(err)
	}
}

func insertSubmittedItem(t *testing.T, ctx context.Context, db queryDB, schema string, item string, target string) {
	t.Helper()

	targetValue := "null"
	if target != "" {
		targetValue = `"` + target + `"`
	}
	_, err := db.ExecContext(ctx, `
INSERT INTO `+schema+`.work_items (id, item_kind_key, payload, submitted_at)
VALUES ($1, 'x08', '{"value":"x75"}', $2)`,
		item,
		fixture.Instant(-4),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO `+schema+`.journal_events (id, coordinate_kind, coordinate_key, event_kind_key, appended_at, payload)
VALUES ('x40' || $1, 'work_item', $1, 'x40', $2, $3)`,
		item,
		fixture.Instant(-4),
		`{"work_item_id":"`+item+`","item_kind":"x08"}`,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO `+schema+`.journal_events (id, coordinate_kind, coordinate_key, event_kind_key, appended_at, payload)
VALUES ('x41' || $1, 'work_item', $1, 'x41', $2, $3)`,
		item,
		fixture.Instant(-4),
		`{"work_item_id":"`+item+`","need_kind":"x12","target_node":`+targetValue+`,"payload":{"value":"x76"}}`,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func journalPayload(
	t *testing.T,
	ctx context.Context,
	db *sql.DB,
	schema string,
	event string,
) map[string]any {
	t.Helper()

	var payload []byte
	err := db.QueryRowContext(ctx, `
SELECT payload
FROM `+schema+`.journal_events
WHERE id = $1`, event).Scan(&payload)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func requirePayloadFields(t *testing.T, body map[string]any, fields map[string]any) {
	t.Helper()

	for key, want := range fields {
		got, ok := body[key]
		if !ok {
			t.Fatalf("payload missing %s in %v", key, body)
		}
		requirePayloadField(t, key, got, want)
	}
}

func requirePayloadField(t *testing.T, key string, got any, want any) {
	t.Helper()

	switch value := want.(type) {
	case []string:
		if !fixture.EqualStrings(payloadStrings(t, key, got), value) {
			t.Fatalf("%s = %v, want %v", key, got, value)
		}
	default:
		if got != want {
			t.Fatalf("%s = %v, want %v", key, got, want)
		}
	}
}

func requirePreconditions(t *testing.T, body map[string]any, expected ...string) {
	t.Helper()

	values := payloadStringSet(t, "preconditions", body["preconditions"])
	for _, value := range expected {
		if !values[value] {
			t.Fatalf("preconditions = %v, missing %s", body["preconditions"], value)
		}
	}
}

func requireNoForbiddenInstructionFields(t *testing.T, body map[string]any) {
	t.Helper()

	for _, key := range []string{
		"actor",
		"role",
		"permission",
		"operator_identity",
		"operator_key",
		"boundary_credential",
		"payload",
		"lease_token",
		"token_digest",
		"sql",
		"storage_error",
		"recommended_action",
	} {
		if _, ok := body[key]; ok {
			t.Fatalf("payload contains forbidden field %s in %v", key, body)
		}
	}
}

func payloadStringSet(t *testing.T, key string, value any) map[string]bool {
	t.Helper()

	values := map[string]bool{}
	for _, item := range payloadStrings(t, key, value) {
		values[item] = true
	}
	return values
}

func payloadStrings(t *testing.T, key string, value any) []string {
	t.Helper()

	raw, ok := value.([]any)
	if !ok {
		t.Fatalf("%s = %T, want array", key, value)
	}
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		text, ok := item.(string)
		if !ok {
			t.Fatalf("%s item = %T, want string", key, item)
		}
		values = append(values, text)
	}
	return values
}

type queryDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}
