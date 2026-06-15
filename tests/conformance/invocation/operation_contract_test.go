package invocation_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestContractNamesBehaviorErrors(t *testing.T) {
	index, err := os.ReadFile(filepath.Join(contractRoot(t), "v1.openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(index)

	for _, name := range behaviorErrorNames() {
		if !strings.Contains(content, name) {
			t.Fatalf("contract index missing behavior error %s", name)
		}
	}
}

func TestContractIndexesEveryCommand(t *testing.T) {
	index, err := os.ReadFile(filepath.Join(contractRoot(t), "v1.openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	for _, command := range commandNames() {
		if !strings.Contains(string(index), "/"+command+":") {
			t.Fatalf("contract index missing command path %s", command)
		}
	}
}

func TestContractIndexesEveryProbe(t *testing.T) {
	index, err := os.ReadFile(filepath.Join(contractRoot(t), "v1.openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	for _, probe := range probeNames() {
		if !strings.Contains(string(index), "/"+probe+":") {
			t.Fatalf("contract index missing probe path %s", probe)
		}
	}
}

func TestContractIndexesEveryMetricRoute(t *testing.T) {
	index, err := os.ReadFile(filepath.Join(contractRoot(t), "v1.openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range metricRouteNames() {
		if !strings.Contains(string(index), "/"+name+":") {
			t.Fatalf("contract index missing metric route %s", name)
		}
	}
}

func TestContractIndexesEveryReadRoute(t *testing.T) {
	index, err := os.ReadFile(filepath.Join(contractRoot(t), "v1.openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(index)
	for _, route := range readRoutes() {
		if !strings.Contains(content, route.path+":") {
			t.Fatalf("contract index missing read route %s", route.path)
		}
	}
}

func TestContractIndexesEveryInstructionRoute(t *testing.T) {
	index, err := os.ReadFile(filepath.Join(contractRoot(t), "v1.openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(index)
	for _, route := range instructionRoutes() {
		if !strings.Contains(content, route.path+":") {
			t.Fatalf("contract index missing instruction route %s", route.path)
		}
	}
}

func commandNames() []string {
	return []string{"submit", "claim", "ack", "nack", "extend", "heartbeat"}
}

func probeNames() []string {
	return []string{"health", "readyz"}
}

func metricRouteNames() []string {
	return []string{"metrics"}
}

type readRoute struct {
	path       string
	operation  string
	schema     string
	aggregate  bool
	parameters []parameterSpec
}

type parameterSpec struct {
	name         string
	location     string
	required     bool
	schemaType   string
	minimum      *int
	maximum      *int
	defaultValue string
	enum         []string
}

type instructionRoute struct {
	path      string
	operation string
	request   string
}

func instructionRoutes() []instructionRoute {
	return []instructionRoute{
		{
			path:      "/operations/instructions/pause",
			operation: "instructionPause",
			request:   "instruction.pause.request.schema.json",
		},
		{
			path:      "/operations/instructions/release-expired-lease",
			operation: "instructionReleaseExpiredLease",
			request:   "instruction.release-expired-lease.request.schema.json",
		},
		{
			path:      "/operations/instructions/force-release-lease",
			operation: "instructionForceReleaseLease",
			request:   "instruction.force-release-lease.request.schema.json",
		},
		{
			path:      "/operations/instructions/move-item",
			operation: "instructionMoveItem",
			request:   "instruction.move-item.request.schema.json",
		},
		{
			path:      "/operations/instructions/move-entries",
			operation: "instructionMoveEntries",
			request:   "instruction.move-entries.request.schema.json",
		},
		{
			path:      "/operations/instructions/move-available",
			operation: "instructionMoveAvailable",
			request:   "instruction.move-available.request.schema.json",
		},
		{
			path:      "/operations/instructions/drop",
			operation: "instructionDrop",
			request:   "instruction.drop.request.schema.json",
		},
		{
			path:      "/operations/instructions/route-outstanding",
			operation: "instructionRouteOutstanding",
			request:   "instruction.route-outstanding.request.schema.json",
		},
	}
}

func readRoutes() []readRoute {
	return []readRoute{
		{path: "/operations", operation: "operations", schema: "operations.response.schema.json", aggregate: true},
		{
			path:      "/operations/channels",
			operation: "operationChannels",
			schema:    "operations.channels.response.schema.json",
			parameters: []parameterSpec{
				limitParameter(50, 200),
				{name: "after_channel_key", location: "query", schemaType: "string"},
				{name: "older_than_seconds", location: "query", schemaType: "integer", minimum: intRef(0)},
			},
		},
		{
			path:      "/operations/channels/{channel_key}/items",
			operation: "operationChannelItems",
			schema:    "operations.channel-items.response.schema.json",
			parameters: []parameterSpec{
				{name: "channel_key", location: "path", required: true, schemaType: "string"},
				limitParameter(50, 100),
				{name: "older_than_seconds", location: "query", schemaType: "integer", minimum: intRef(0)},
				{
					name:         "lease_view",
					location:     "query",
					schemaType:   "string",
					defaultValue: "all",
					enum:         []string{"all", "expired", "held", "none"},
				},
			},
		},
		{
			path:      "/operations/items/{work_item_id}",
			operation: "operationItem",
			schema:    "operations.item.response.schema.json",
			parameters: []parameterSpec{
				{name: "work_item_id", location: "path", required: true, schemaType: "string"},
			},
		},
		{
			path:      "/operations/items/{work_item_id}/journal",
			operation: "operationItemJournal",
			schema:    "operations.item-journal.response.schema.json",
			parameters: []parameterSpec{
				{name: "work_item_id", location: "path", required: true, schemaType: "string"},
				limitParameter(50, 200),
				{name: "after_append_index", location: "query", schemaType: "integer", minimum: intRef(0)},
			},
		},
		{
			path:      "/operations/nodes",
			operation: "operationNodes",
			schema:    "operations.nodes.response.schema.json",
			parameters: []parameterSpec{
				limitParameter(100, 200),
				{name: "after_node_key", location: "query", schemaType: "string"},
				{name: "need_kind", location: "query", schemaType: "string"},
			},
		},
	}
}

func limitParameter(defaultValue int, maximum int) parameterSpec {
	return parameterSpec{
		name:         "limit",
		location:     "query",
		schemaType:   "integer",
		minimum:      intRef(1),
		maximum:      intRef(maximum),
		defaultValue: stringFromInt(defaultValue),
	}
}

func intRef(value int) *int {
	return &value
}

func stringFromInt(value int) string {
	return strconv.Itoa(value)
}

func behaviorErrorNames() []string {
	return []string{
		"invalid_input",
		"unknown_vocabulary",
		"empty_queue",
		"invalid_lease",
		"expired_lease",
		"no_route",
		"conflict",
	}
}
