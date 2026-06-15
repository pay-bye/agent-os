package invocation_test

import (
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/tests/conformance/schemadoc"
)

func TestContractAcceptsSyntheticCommandPayloads(t *testing.T) {
	for _, test := range validPayloadCases() {
		t.Run(test.name, func(t *testing.T) {
			schema := readSchema(t, test.schema)

			if err := schemadoc.Validate(schema, test.payload); err != nil {
				t.Fatalf("valid payload rejected: %v", err)
			}
		})
	}
}

func TestContractRejectsMissingRequiredFields(t *testing.T) {
	for _, test := range missingFieldCases() {
		t.Run(test.name, func(t *testing.T) {
			schema := readSchema(t, test.schema)

			err := schemadoc.Validate(schema, test.payload)

			if err == nil {
				t.Fatal("expected missing required field rejection")
			}
			if !strings.Contains(err.Error(), test.field) {
				t.Fatalf("error = %v, want missing %s", err, test.field)
			}
		})
	}
}

func TestContractRejectsMalformedNeedPayloads(t *testing.T) {
	requireMalformedPayloadsRejected(t, malformedNeedPayloadCases())
}

func TestContractRejectsMalformedResponseVariants(t *testing.T) {
	requireMalformedPayloadsRejected(t, malformedResponseVariantCases())
}

func TestContractRejectsMalformedCompatibilityPayloads(t *testing.T) {
	requireMalformedPayloadsRejected(t, malformedCompatibilityPayloadCases())
}

func TestContractRejectsMalformedProbePayloads(t *testing.T) {
	requireMalformedPayloadsRejected(t, malformedProbePayloadCases())
}

func TestContractRejectsMalformedOperationsPayloads(t *testing.T) {
	requireMalformedPayloadsRejected(t, malformedOperationsPayloadCases())
}

func TestContractRejectsMalformedInstructionPayloads(t *testing.T) {
	requireMalformedPayloadsRejected(t, malformedInstructionPayloadCases())
}

func requireMalformedPayloadsRejected(t *testing.T, cases []payloadCase) {
	t.Helper()

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			schema := readSchema(t, test.schema)

			err := schemadoc.Validate(schema, test.payload)

			if err == nil {
				t.Fatal("expected malformed payload rejection")
			}
			if !strings.Contains(err.Error(), test.field) {
				t.Fatalf("error = %v, want %s", err, test.field)
			}
		})
	}
}

type payloadCase struct {
	name    string
	schema  string
	payload map[string]any
	field   string
}

func validPayloadCases() []payloadCase {
	return []payloadCase{
		{name: "submit request", schema: "submit.request.schema.json", payload: submitPayload()},
		{name: "claim request", schema: "claim.request.schema.json", payload: claimPayload()},
		{name: "ack request", schema: "ack.request.schema.json", payload: resolutionPayload()},
		{name: "nack request", schema: "nack.request.schema.json", payload: resolutionPayload()},
		{name: "extend request", schema: "extend.request.schema.json", payload: extendPayload()},
		{name: "heartbeat request", schema: "heartbeat.request.schema.json", payload: heartbeatPayload()},
		{name: "health response", schema: "health.response.schema.json", payload: healthResponse()},
		{name: "readyz ready response", schema: "readyz.response.schema.json", payload: readyzReadyResponse()},
		{name: "readyz not ready response", schema: "readyz.response.schema.json", payload: readyzNotReadyResponse()},
		{name: "operations response", schema: "operations.response.schema.json", payload: operationsResponse()},
		{name: "operations channels response", schema: "operations.channels.response.schema.json", payload: operationsChannelsResponse()},
		{name: "operations channel items response", schema: "operations.channel-items.response.schema.json", payload: operationsChannelItemsResponse()},
		{name: "operations item response", schema: "operations.item.response.schema.json", payload: operationsItemResponse()},
		{name: "operations item journal response", schema: "operations.item-journal.response.schema.json", payload: operationsItemJournalResponse()},
		{name: "operations nodes response", schema: "operations.nodes.response.schema.json", payload: operationsNodesResponse()},
		{name: "instruction pause request", schema: "instruction.pause.request.schema.json", payload: instructionPausePayload()},
		{name: "instruction release expired lease request", schema: "instruction.release-expired-lease.request.schema.json", payload: instructionLeasePayload()},
		{name: "instruction force release lease request", schema: "instruction.force-release-lease.request.schema.json", payload: instructionLeasePayload()},
		{name: "instruction move item request", schema: "instruction.move-item.request.schema.json", payload: instructionMoveItemPayload()},
		{name: "instruction move entries request", schema: "instruction.move-entries.request.schema.json", payload: instructionMoveEntriesPayload()},
		{name: "instruction move available request", schema: "instruction.move-available.request.schema.json", payload: instructionMoveAvailablePayload()},
		{name: "instruction drop request", schema: "instruction.drop.request.schema.json", payload: instructionItemsPayload()},
		{name: "instruction route outstanding request", schema: "instruction.route-outstanding.request.schema.json", payload: instructionItemsPayload()},
		{name: "instruction applied response", schema: "instruction.response.schema.json", payload: instructionAppliedResponse()},
		{name: "instruction precondition response", schema: "instruction.response.schema.json", payload: instructionPreconditionResponse()},
		{name: "submit routed response", schema: "submit.response.schema.json", payload: routedSubmitResponse()},
		{name: "submit waiting response", schema: "submit.response.schema.json", payload: waitingSubmitResponse()},
		{name: "claim empty response", schema: "claim.response.schema.json", payload: emptyClaimResponse()},
		{name: "claim lease response", schema: "claim.response.schema.json", payload: leaseClaimResponse()},
		{name: "ack routed response", schema: "ack.response.schema.json", payload: routedResolutionResponse()},
		{name: "ack complete response", schema: "ack.response.schema.json", payload: completeResolutionResponse()},
		{name: "nack routed response", schema: "nack.response.schema.json", payload: routedResolutionResponse()},
		{name: "nack complete response", schema: "nack.response.schema.json", payload: completeResolutionResponse()},
		{name: "compatibility response", schema: "compatibility.response.schema.json", payload: compatibilityResponse()},
	}
}

func missingFieldCases() []payloadCase {
	return []payloadCase{
		{name: "submit identity", schema: "submit.request.schema.json", payload: without(submitPayload(), "work_item_id"), field: "work_item_id"},
		{name: "claim channel", schema: "claim.request.schema.json", payload: without(claimPayload(), "channel_key"), field: "channel_key"},
		{name: "ack lease", schema: "ack.request.schema.json", payload: without(resolutionPayload(), "lease_id"), field: "lease_id"},
		{name: "ack token", schema: "ack.request.schema.json", payload: without(resolutionPayload(), "lease_token"), field: "lease_token"},
		{name: "nack token", schema: "nack.request.schema.json", payload: without(resolutionPayload(), "lease_token"), field: "lease_token"},
		{name: "extend token", schema: "extend.request.schema.json", payload: without(extendPayload(), "lease_token"), field: "lease_token"},
		{name: "heartbeat token", schema: "heartbeat.request.schema.json", payload: without(heartbeatPayload(), "lease_token"), field: "lease_token"},
		{name: "extend expiry", schema: "extend.request.schema.json", payload: without(extendPayload(), "requested_expires_at"), field: "requested_expires_at"},
		{name: "heartbeat lease", schema: "heartbeat.request.schema.json", payload: without(heartbeatPayload(), "lease_id"), field: "lease_id"},
		{name: "health result", schema: "health.response.schema.json", payload: without(healthResponse(), "result"), field: "result"},
		{name: "readyz checks", schema: "readyz.response.schema.json", payload: without(readyzReadyResponse(), "checks"), field: "checks"},
		{name: "operations views", schema: "operations.response.schema.json", payload: without(operationsResponse(), "views"), field: "views"},
		{name: "instruction identity", schema: "instruction.pause.request.schema.json", payload: without(instructionPausePayload(), "instruction_id"), field: "instruction_id"},
		{name: "compatibility digest", schema: "compatibility.response.schema.json", payload: without(compatibilityResponse(), "schema_set_digest"), field: "schema_set_digest"},
	}
}

func malformedNeedPayloadCases() []payloadCase {
	return []payloadCase{
		{
			name:    "declared need item shape",
			schema:  "submit.request.schema.json",
			payload: with(submitPayload(), "declared_needs", []any{"x04"}),
			field:   "declared_needs[0]",
		},
		{
			name:   "declared need kind",
			schema: "ack.request.schema.json",
			payload: with(
				resolutionPayload(),
				"declared_needs",
				[]any{map[string]any{"payload": map[string]any{"value": "x45"}}},
			),
			field: "need_kind",
		},
		{
			name:   "declared need empty target",
			schema: "nack.request.schema.json",
			payload: with(
				resolutionPayload(),
				"declared_needs",
				[]any{map[string]any{"need_kind": "x05", "target_node": ""}},
			),
			field: "target_node",
		},
	}
}

func malformedResponseVariantCases() []payloadCase {
	return []payloadCase{
		{
			name:    "claim lease variant",
			schema:  "claim.response.schema.json",
			payload: map[string]any{"empty": false},
			field:   "lease_id",
		},
		{
			name:    "submit route variant",
			schema:  "submit.response.schema.json",
			payload: map[string]any{"work_item_id": "x02", "routed": true},
			field:   "channel_key",
		},
		{
			name:    "ack route variant",
			schema:  "ack.response.schema.json",
			payload: map[string]any{"resolved": true, "routed": true},
			field:   "channel_key",
		},
		{
			name:    "nack route variant",
			schema:  "nack.response.schema.json",
			payload: map[string]any{"resolved": true, "routed": true},
			field:   "channel_key",
		},
	}
}

func malformedCompatibilityPayloadCases() []payloadCase {
	return []payloadCase{
		{
			name:    "compatibility extra field",
			schema:  "compatibility.response.schema.json",
			payload: with(compatibilityResponse(), "extra", "x91"),
			field:   "extra",
		},
		{
			name:    "compatibility digest shape",
			schema:  "compatibility.response.schema.json",
			payload: with(compatibilityResponse(), "schema_set_digest", "x91"),
			field:   "schema_set_digest",
		},
		{
			name:   "compatibility route inventory",
			schema: "compatibility.response.schema.json",
			payload: with(
				compatibilityResponse(),
				"routes",
				[]any{map[string]any{"method": "GET", "path": "/x91"}},
			),
			field: "routes[0]",
		},
	}
}

func malformedProbePayloadCases() []payloadCase {
	return []payloadCase{
		{
			name:    "health extra field",
			schema:  "health.response.schema.json",
			payload: with(healthResponse(), "checks", map[string]any{"storage": "ready"}),
			field:   "checks",
		},
		{
			name:    "health result value",
			schema:  "health.response.schema.json",
			payload: with(healthResponse(), "result", "ready"),
			field:   "result",
		},
		{
			name:    "readyz top result",
			schema:  "readyz.response.schema.json",
			payload: with(readyzReadyResponse(), "result", "live"),
			field:   "result",
		},
		{
			name:    "readyz missing check",
			schema:  "readyz.response.schema.json",
			payload: with(readyzReadyResponse(), "checks", without(readyChecks(), "storage")),
			field:   "storage",
		},
		{
			name:    "readyz check raw detail",
			schema:  "readyz.response.schema.json",
			payload: with(readyzReadyResponse(), "checks", with(readyChecks(), "storage", "postgres://secret")),
			field:   "storage",
		},
		{
			name:    "readyz extra field",
			schema:  "readyz.response.schema.json",
			payload: with(readyzReadyResponse(), "detail", "database_url"),
			field:   "detail",
		},
	}
}

func malformedOperationsPayloadCases() []payloadCase {
	return []payloadCase{
		{
			name:    "operations extra field",
			schema:  "operations.response.schema.json",
			payload: with(operationsResponse(), "extra", "x91"),
			field:   "extra",
		},
		{
			name:    "operations result value",
			schema:  "operations.response.schema.json",
			payload: with(operationsResponse(), "result", "ready"),
			field:   "result",
		},
		{
			name:   "operations unknown unavailable group",
			schema: "operations.response.schema.json",
			payload: with(
				operationsResponse(),
				"unavailable",
				[]any{"queue", "x91"},
			),
			field: "unavailable[1]",
		},
		{
			name:   "operations unknown view group",
			schema: "operations.response.schema.json",
			payload: with(
				operationsResponse(),
				"views",
				with(operationsViews(), "extra", map[string]any{}),
			),
			field: "extra",
		},
		{
			name:   "operations queue raw field",
			schema: "operations.response.schema.json",
			payload: with(
				operationsResponse(),
				"views",
				with(
					operationsViews(),
					"queue",
					with(queueView(), "channel_key", "x91"),
				),
			),
			field: "channel_key",
		},
		{
			name:   "operations compatibility feature value",
			schema: "operations.response.schema.json",
			payload: with(
				operationsResponse(),
				"views",
				with(
					operationsViews(),
					"compatibility",
					with(operationsCompatibilityView(), "features", featuresWithInvalidValue()),
				),
			),
			field: "features[6]",
		},
		{
			name:   "operations compatibility feature budget",
			schema: "operations.response.schema.json",
			payload: with(
				operationsResponse(),
				"views",
				with(
					operationsViews(),
					"compatibility",
					with(operationsCompatibilityView(), "features", overBudgetFeatures()),
				),
			),
			field: "features",
		},
		{
			name:   "operations compatibility route value",
			schema: "operations.response.schema.json",
			payload: with(
				operationsResponse(),
				"views",
				with(
					operationsViews(),
					"compatibility",
					with(operationsCompatibilityView(), "routes", routesWithInvalidValue()),
				),
			),
			field: "routes[23]",
		},
		{
			name:   "operations compatibility route budget",
			schema: "operations.response.schema.json",
			payload: with(
				operationsResponse(),
				"views",
				with(
					operationsViews(),
					"compatibility",
					with(operationsCompatibilityView(), "routes", overBudgetRoutes()),
				),
			),
			field: "routes",
		},
		{
			name:    "operations channels payload field",
			schema:  "operations.channels.response.schema.json",
			payload: with(operationsChannelsResponse(), "payload", map[string]any{}),
			field:   "payload",
		},
		{
			name:   "operations channel items lease token",
			schema: "operations.channel-items.response.schema.json",
			payload: with(
				operationsChannelItemsResponse(),
				"items",
				[]any{with(channelItem(), "lease", with(leaseView(), "lease_token", "x91"))},
			),
			field: "lease_token",
		},
		{
			name:    "operations item status",
			schema:  "operations.item.response.schema.json",
			payload: with(operationsItemResponse(), "status", "healthy"),
			field:   "status",
		},
		{
			name:   "operations item nested work item",
			schema: "operations.item.response.schema.json",
			payload: with(
				operationsItemResponse(),
				"channel_entry",
				with(itemChannelEntry(), "work_item_id", "x08"),
			),
			field: "work_item_id",
		},
		{
			name:   "operations item journal raw payload",
			schema: "operations.item-journal.response.schema.json",
			payload: with(
				operationsItemJournalResponse(),
				"events",
				[]any{with(journalEvent(), "metadata", with(journalMetadata(), "payload", map[string]any{"value": "x91"}))},
			),
			field: "payload",
		},
		{
			name:   "operations nodes verdict field",
			schema: "operations.nodes.response.schema.json",
			payload: with(
				operationsNodesResponse(),
				"nodes",
				[]any{with(nodeView(), "drift_verdict", "matching")},
			),
			field: "drift_verdict",
		},
	}
}

func malformedInstructionPayloadCases() []payloadCase {
	return []payloadCase{
		{
			name:    "pause payload rejected",
			schema:  "instruction.pause.request.schema.json",
			payload: with(instructionPausePayload(), "payload", map[string]any{"value": "x91"}),
			field:   "payload",
		},
		{
			name:    "lease token rejected",
			schema:  "instruction.force-release-lease.request.schema.json",
			payload: with(instructionLeasePayload(), "lease_token", "x91"),
			field:   "lease_token",
		},
		{
			name:    "role rejected",
			schema:  "instruction.move-item.request.schema.json",
			payload: with(instructionMoveItemPayload(), "role", "x91"),
			field:   "role",
		},
		{
			name:    "duplicate entries rejected",
			schema:  "instruction.move-entries.request.schema.json",
			payload: with(instructionMoveEntriesPayload(), "entry_ids", []any{"x31", "x31"}),
			field:   "entry_ids",
		},
		{
			name:    "move available zero limit rejected",
			schema:  "instruction.move-available.request.schema.json",
			payload: with(instructionMoveAvailablePayload(), "limit", float64(0)),
			field:   "limit",
		},
		{
			name:    "drop over budget rejected",
			schema:  "instruction.drop.request.schema.json",
			payload: with(instructionItemsPayload(), "work_item_ids", overBudgetIDs()),
			field:   "work_item_ids",
		},
		{
			name:    "response token material rejected",
			schema:  "instruction.response.schema.json",
			payload: with(instructionAppliedResponse(), "lease_token", "x91"),
			field:   "lease_token",
		},
	}
}

func submitPayload() map[string]any {
	return map[string]any{
		"work_item_id": "x02",
		"item_kind":    "x03",
		"payload":      map[string]any{"value": "x45"},
		"declared_needs": []any{
			map[string]any{
				"need_kind":   "x04",
				"target_node": "x09",
				"payload":     map[string]any{"target_node": "x98", "value": "x45"},
			},
		},
	}
}

func instructionPausePayload() map[string]any {
	return map[string]any{
		"instruction_id": "x70",
		"node_key":       "x17",
	}
}

func instructionLeasePayload() map[string]any {
	return map[string]any{
		"instruction_id": "x70",
		"lease_id":       "x16",
	}
}

func instructionMoveItemPayload() map[string]any {
	return map[string]any{
		"instruction_id":     "x70",
		"work_item_id":       "x08",
		"source_channel_key": "x15",
		"target_channel_key": "x68",
	}
}

func instructionMoveEntriesPayload() map[string]any {
	return map[string]any{
		"instruction_id":     "x70",
		"source_channel_key": "x15",
		"target_channel_key": "x68",
		"entry_ids":          []any{"x31", "x32"},
	}
}

func instructionMoveAvailablePayload() map[string]any {
	return map[string]any{
		"instruction_id":     "x70",
		"source_channel_key": "x15",
		"target_channel_key": "x68",
		"limit":              float64(2),
	}
}

func instructionItemsPayload() map[string]any {
	return map[string]any{
		"instruction_id": "x70",
		"work_item_ids":  []any{"x08", "x09"},
	}
}

func instructionAppliedResponse() map[string]any {
	return map[string]any{
		"instruction_id": "x70",
		"result":         "applied",
		"event_ids":      []any{"x80"},
		"affected_count": float64(1),
		"affected_ids":   []any{"x08"},
	}
}

func instructionPreconditionResponse() map[string]any {
	return map[string]any{
		"instruction_id":      "x70",
		"result":              "precondition_failed",
		"event_ids":           []any{"x80"},
		"affected_count":      float64(0),
		"affected_ids":        []any{},
		"failed_precondition": "lease_expired",
	}
}

func overBudgetIDs() []any {
	values := make([]any, 101)
	for index := range values {
		values[index] = "x91"
	}
	return values
}

func claimPayload() map[string]any {
	return map[string]any{
		"channel_key":   "x06",
		"lease_id":      "x07",
		"lease_seconds": float64(60),
	}
}

func resolutionPayload() map[string]any {
	return map[string]any{
		"lease_id":    "x07",
		"lease_token": "x08",
		"declared_needs": []any{
			map[string]any{
				"need_kind":   "x05",
				"target_node": "x09",
				"payload":     map[string]any{"target_node": "x98", "value": "x45"},
			},
		},
	}
}

func extendPayload() map[string]any {
	return map[string]any{
		"lease_id":             "x07",
		"lease_token":          "x08",
		"requested_expires_at": "2026-05-18T12:10:00Z",
	}
}

func heartbeatPayload() map[string]any {
	return map[string]any{"lease_id": "x07", "lease_token": "x08"}
}

func healthResponse() map[string]any {
	return map[string]any{"result": "live"}
}

func readyzReadyResponse() map[string]any {
	return map[string]any{
		"result": "ready",
		"checks": readyChecks(),
	}
}

func readyzNotReadyResponse() map[string]any {
	checks := readyChecks()
	checks["storage"] = "not_ready"
	return map[string]any{
		"result": "not_ready",
		"checks": checks,
	}
}

func readyChecks() map[string]any {
	return map[string]any{
		"startup":     "ready",
		"storage":     "ready",
		"migrations":  "ready",
		"verifier":    "ready",
		"declaration": "ready",
		"handler":     "ready",
	}
}

func operationsResponse() map[string]any {
	return map[string]any{
		"generated_at":   "2026-05-18T12:00:00Z",
		"result":         "complete",
		"window_seconds": float64(300),
		"views":          operationsViews(),
		"unavailable":    []any{},
	}
}

func operationsViews() map[string]any {
	return map[string]any{
		"queue":         queueView(),
		"leases":        leasesView(),
		"journal":       journalView(),
		"commands":      commandsView(),
		"routing":       routingView(),
		"build":         buildView(),
		"compatibility": operationsCompatibilityView(),
	}
}

func operationsCompatibilityView() map[string]any {
	body := compatibilityResponse()
	return map[string]any{
		"contract_version": body["contract_version"],
		"features":         body["features"],
		"routes":           body["routes"],
	}
}

func featuresWithInvalidValue() []any {
	features := compatibilityFeatures()
	features[len(features)-1] = "x91"
	return features
}

func overBudgetFeatures() []any {
	features := compatibilityFeatures()
	return append(features, "lease_claim")
}

func routesWithInvalidValue() []any {
	routes := compatibilityRoutes()
	routes[len(routes)-1] = map[string]any{"method": "GET", "path": "/x91"}
	return routes
}

func overBudgetRoutes() []any {
	routes := compatibilityRoutes()
	return append(routes, map[string]any{"method": "GET", "path": "/operations"})
}

func queueView() map[string]any {
	return map[string]any{
		"channel_class":                "all",
		"depth":                        float64(12),
		"available":                    float64(7),
		"oldest_available_age_seconds": float64(84),
	}
}

func leasesView() map[string]any {
	return map[string]any{
		"channel_class": "all",
		"held":          float64(3),
		"expired":       float64(1),
	}
}

func journalView() map[string]any {
	return map[string]any{
		"append_rate_per_second": float64(0.4),
		"window_seconds":         float64(300),
	}
}

func commandsView() map[string]any {
	return map[string]any{
		"succeeded": float64(40),
		"failed":    float64(2),
	}
}

func routingView() map[string]any {
	return map[string]any{
		"routed":   float64(38),
		"unrouted": float64(4),
	}
}

func buildView() map[string]any {
	return map[string]any{
		"version":  "v1.2.3",
		"revision": "a1b2c3",
	}
}

func routedSubmitResponse() map[string]any {
	return map[string]any{
		"work_item_id": "x02",
		"routed":       true,
		"channel_key":  "x06",
	}
}

func waitingSubmitResponse() map[string]any {
	return map[string]any{
		"work_item_id": "x02",
		"routed":       false,
	}
}

func emptyClaimResponse() map[string]any {
	return map[string]any{"empty": true}
}

func leaseClaimResponse() map[string]any {
	return map[string]any{
		"empty":        false,
		"lease_id":     "x07",
		"lease_token":  "x08",
		"work_item_id": "x02",
		"payload":      map[string]any{"value": "x45"},
		"expires_at":   "2026-05-18T12:10:00Z",
	}
}

func routedResolutionResponse() map[string]any {
	return map[string]any{
		"resolved":    true,
		"routed":      true,
		"channel_key": "x06",
	}
}

func completeResolutionResponse() map[string]any {
	return map[string]any{
		"resolved": true,
		"routed":   false,
	}
}

func operationsChannelsResponse() map[string]any {
	return map[string]any{
		"channels": []any{
			map[string]any{
				"channel_key":                  "x15",
				"node_key":                     "x17",
				"depth":                        float64(2),
				"available":                    float64(1),
				"oldest_available_age_seconds": float64(120),
			},
		},
	}
}

func operationsChannelItemsResponse() map[string]any {
	return map[string]any{"items": []any{channelItem()}}
}

func channelItem() map[string]any {
	return map[string]any{
		"entry_id":     "x01",
		"work_item_id": "x08",
		"channel_key":  "x15",
		"node_key":     "x17",
		"enqueued_at":  "2026-05-18T11:58:00Z",
		"available_at": "2026-05-18T11:59:00Z",
		"age_seconds":  float64(60),
		"lease":        leaseView(),
	}
}

func leaseView() map[string]any {
	return map[string]any{
		"lease_id":   "x13",
		"granted_at": "2026-05-18T12:00:00Z",
		"expires_at": "2026-05-18T12:01:00Z",
	}
}

func operationsItemResponse() map[string]any {
	return map[string]any{
		"work_item_id":  "x08",
		"item_kind":     "x03",
		"submitted_at":  "2026-05-18T11:55:00Z",
		"channel_entry": itemChannelEntry(),
		"lease": map[string]any{
			"lease_id":    "x13",
			"channel_key": "x15",
			"granted_at":  "2026-05-18T12:00:00Z",
			"expires_at":  "2026-05-18T12:01:00Z",
		},
		"outstanding_need": map[string]any{
			"event_id":    "x21",
			"need_kind":   "x12",
			"target_node": "x17",
			"declared_at": "2026-05-18T11:59:30Z",
		},
	}
}

func itemChannelEntry() map[string]any {
	return map[string]any{
		"entry_id":     "x01",
		"channel_key":  "x15",
		"node_key":     "x17",
		"enqueued_at":  "2026-05-18T11:58:00Z",
		"available_at": "2026-05-18T11:59:00Z",
		"age_seconds":  float64(60),
	}
}

func operationsItemJournalResponse() map[string]any {
	return map[string]any{"events": []any{journalEvent()}}
}

func journalEvent() map[string]any {
	return map[string]any{
		"event_id":     "x21",
		"event_kind":   "x41",
		"appended_at":  "2026-05-18T12:00:00Z",
		"append_index": float64(1),
		"metadata":     journalMetadata(),
	}
}

func journalMetadata() map[string]any {
	return map[string]any{
		"work_item_id": "x08",
		"need_kind":    "x12",
	}
}

func operationsNodesResponse() map[string]any {
	return map[string]any{"nodes": []any{nodeView()}}
}

func nodeView() map[string]any {
	return map[string]any{
		"node_key":    "x17",
		"channel_key": "x15",
		"need_kinds":  []any{"x12"},
		"routable":    false,
	}
}

func compatibilityResponse() map[string]any {
	return map[string]any{
		"contract_version":  "v1",
		"schema_set_digest": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		"features":          compatibilityFeatures(),
		"routes":            compatibilityRoutes(),
	}
}

func compatibilityFeatures() []any {
	return []any{
		"lease_claim",
		"lease_extend",
		"lease_ack",
		"lease_nack",
		"lease_capability",
		"declared_needs",
		"failure_payload",
	}
}

func compatibilityRoutes() []any {
	return []any{
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
		map[string]any{"method": "GET", "path": "/compatibility"},
		map[string]any{"method": "GET", "path": "/operations"},
		map[string]any{"method": "GET", "path": "/operations/channels"},
		map[string]any{"method": "GET", "path": "/operations/channels/{channel_key}/items"},
		map[string]any{"method": "GET", "path": "/operations/items/{work_item_id}"},
		map[string]any{"method": "GET", "path": "/operations/items/{work_item_id}/journal"},
		map[string]any{"method": "GET", "path": "/operations/nodes"},
	}
}

func without(payload map[string]any, field string) map[string]any {
	copy := map[string]any{}
	for key, value := range payload {
		if key != field {
			copy[key] = value
		}
	}
	return copy
}

func with(payload map[string]any, field string, value any) map[string]any {
	copy := copyPayload(payload)
	copy[field] = value
	return copy
}

func copyPayload(payload map[string]any) map[string]any {
	copy := map[string]any{}
	for key, value := range payload {
		copy[key] = value
	}
	return copy
}
