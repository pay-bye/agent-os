package postgres

import (
	"reflect"
	"testing"
)

func TestDecodeSchemas(t *testing.T) {
	schemas, err := DecodeSchemas(&rowsValues{rows: [][]any{{"x01", []byte(`{"type":"object"}`)}}})
	if err != nil {
		t.Fatal(err)
	}

	requireEqual(t, schemas, []SchemaRecord{{Key: "x01", Document: []byte(`{"type":"object"}`)}})
}

func TestDecodeItems(t *testing.T) {
	items, err := DecodeItems(&rowsValues{rows: [][]any{{"x08", "x01", "item"}}})
	if err != nil {
		t.Fatal(err)
	}

	requireEqual(t, items, []ItemRecord{{Key: "x08", Schema: "x01", Description: "item"}})
}

func TestDecodeNeeds(t *testing.T) {
	needs, err := DecodeNeeds(&rowsValues{rows: [][]any{{"x12", "x02", "need"}, {"x13", nil, "open"}}})
	if err != nil {
		t.Fatal(err)
	}

	requireEqual(t, needs, []NeedRecord{
		{Key: "x12", Schema: "x02", HasSchema: true, Description: "need"},
		{Key: "x13", Description: "open"},
	})
}

func TestDecodeNodes(t *testing.T) {
	nodes, err := DecodeNodes(&rowsValues{rows: [][]any{{"x17", "node", "x15", "channel"}}})
	if err != nil {
		t.Fatal(err)
	}

	requireEqual(t, nodes, []NodeRecord{{Key: "x17", Description: "node", Channel: "x15", ChannelLabel: "channel"}})
}

func TestDecodeAccepts(t *testing.T) {
	accepts, err := DecodeAccepts(&rowsValues{rows: [][]any{{"x12"}, {"x13"}}})
	if err != nil {
		t.Fatal(err)
	}

	requireEqual(t, accepts, []string{"x12", "x13"})
}

func TestDecodeRoutes(t *testing.T) {
	routes, err := DecodeRoutes(&rowsValues{rows: [][]any{{"x12", "x17", 1}}})
	if err != nil {
		t.Fatal(err)
	}

	requireEqual(t, routes, []RouteRecord{{Need: "x12", Node: "x17", Order: 1}})
}

func TestDecodeRoutingExclusions(t *testing.T) {
	exclusions, err := DecodeRoutingExclusions(&rowsValues{rows: [][]any{{"x17"}}})
	if err != nil {
		t.Fatal(err)
	}

	requireEqual(t, exclusions, []RoutingExclusionRecord{{Node: "x17"}})
}

func requireEqual[T any](t *testing.T, got, want T) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
