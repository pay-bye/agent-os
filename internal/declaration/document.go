package declaration

import (
	"errors"
	"fmt"
	"github.com/pay-bye/agent-os/internal/storage/postgres"
	"sort"
)

const DefaultPath = "vocabulary.yaml"

var ErrInvalid = errors.New("invalid declaration")

type Document struct {
	Version int
	Schemas map[string]Schema
	Items   map[string]Item
	Needs   map[string]Need
	Nodes   map[string]Node
	Routes  map[string][]Route
}

func (d Document) Vocabulary() postgres.Catalog {
	return postgres.Catalog{
		Schemas: schemaRecords(d.Schemas),
		Items:   itemRecords(d.Items),
		Needs:   needRecords(d.Needs),
		Nodes:   nodeRecords(d.Nodes),
		Routes:  routeRecords(d.Routes),
	}
}

type Schema struct {
	Document []byte
}

type Item struct {
	Schema      string
	Description string
}

type Need struct {
	Schema      string
	HasSchema   bool
	Description string
}

type Node struct {
	Description string
	Accepts     []string
}

type Route struct {
	Node string
}

func schemaRecords(items map[string]Schema) []postgres.SchemaRecord {
	keys := sortedKeys(items)
	records := make([]postgres.SchemaRecord, 0, len(keys))
	for _, key := range keys {
		records = append(records, postgres.SchemaRecord{Key: key, Document: items[key].Document})
	}
	return records
}

func itemRecords(items map[string]Item) []postgres.ItemRecord {
	keys := sortedKeys(items)
	records := make([]postgres.ItemRecord, 0, len(keys))
	for _, key := range keys {
		item := items[key]
		records = append(records, postgres.ItemRecord{
			Key: key, Schema: item.Schema, Description: item.Description,
		})
	}
	return records
}

func needRecords(items map[string]Need) []postgres.NeedRecord {
	keys := sortedKeys(items)
	records := make([]postgres.NeedRecord, 0, len(keys))
	for _, key := range keys {
		need := items[key]
		records = append(records, postgres.NeedRecord{
			Key: key, Schema: need.Schema, HasSchema: need.HasSchema, Description: need.Description,
		})
	}
	return records
}

func nodeRecords(items map[string]Node) []postgres.NodeRecord {
	keys := sortedKeys(items)
	records := make([]postgres.NodeRecord, 0, len(keys))
	for _, key := range keys {
		node := items[key]
		records = append(records, postgres.NodeRecord{
			Key:          key,
			Description:  node.Description,
			Accepts:      append([]string(nil), node.Accepts...),
			Channel:      key,
			ChannelLabel: node.Description,
		})
	}
	return records
}

func routeRecords(items map[string][]Route) []postgres.RouteRecord {
	keys := sortedKeys(items)
	var records []postgres.RouteRecord
	for _, need := range keys {
		for index, route := range items[need] {
			records = append(records, postgres.RouteRecord{Need: need, Node: route.Node, Order: index + 1})
		}
	}
	return records
}

func sortedKeys[T any](items map[string]T) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func invalid(code string, detail string) error {
	return fmt.Errorf("%w: %s: %s", ErrInvalid, code, detail)
}
