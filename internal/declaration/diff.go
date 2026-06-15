package declaration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pay-bye/agent-os/internal/storage/postgres"
	"sort"
)

func BuildDelta(current postgres.Catalog, desired postgres.Catalog) Delta {
	delta := Delta{Installable: true}
	delta = compareRecords(delta, "schema", schemaIndex(current.Schemas), schemaIndex(desired.Schemas))
	delta = compareRecords(delta, "item", itemIndex(current.Items), itemIndex(desired.Items))
	delta = compareRecords(delta, "need", needIndex(current.Needs), needIndex(desired.Needs))
	delta = compareRecords(delta, "node", nodeIndex(current.Nodes), nodeIndex(desired.Nodes))
	delta = compareRecords(delta, "route", routeIndex(current.Routes), routeIndex(desired.Routes))
	delta = compareClearances(delta, current.RoutingExclusions, nodeIndex(desired.Nodes))
	sortRemovals(delta.Removals)
	delta.Installable = len(delta.Conflicts) == 0
	return delta
}

func compareRecords(delta Delta, kind string, current map[string][]byte, desired map[string][]byte) Delta {
	for _, key := range sortedBytesKeys(desired) {
		currentValue, ok := current[key]
		switch {
		case !ok:
			delta.Additions = append(delta.Additions, RecordRef{Kind: kind, Key: key})
		case !bytes.Equal(currentValue, desired[key]):
			delta.Conflicts = append(delta.Conflicts, RecordConflict{Kind: kind, Key: key, State: "different"})
		}
	}
	for _, key := range sortedBytesKeys(current) {
		if _, ok := desired[key]; !ok {
			delta.Removals = append(delta.Removals, RecordRef{Kind: kind, Key: key})
		}
	}
	return delta
}

func compareClearances(
	delta Delta,
	current []postgres.RoutingExclusionRecord,
	desiredNodes map[string][]byte,
) Delta {
	for _, exclusion := range current {
		if _, ok := desiredNodes[exclusion.Node]; ok {
			delta.Clearances = append(delta.Clearances, RecordRef{Kind: "routing_exclusion", Key: exclusion.Node})
		}
	}
	return delta
}

func schemaIndex(records []postgres.SchemaRecord) map[string][]byte {
	items := map[string][]byte{}
	for _, record := range records {
		items[record.Key] = normalizeJSON(record.Document)
	}
	return items
}

func itemIndex(records []postgres.ItemRecord) map[string][]byte {
	items := map[string][]byte{}
	for _, record := range records {
		items[record.Key] = encoded(record)
	}
	return items
}

func needIndex(records []postgres.NeedRecord) map[string][]byte {
	items := map[string][]byte{}
	for _, record := range records {
		items[record.Key] = encoded(record)
	}
	return items
}

func nodeIndex(records []postgres.NodeRecord) map[string][]byte {
	items := map[string][]byte{}
	for _, record := range records {
		sorted := record
		sort.Strings(sorted.Accepts)
		items[record.Key] = encoded(sorted)
	}
	return items
}

func routeIndex(records []postgres.RouteRecord) map[string][]byte {
	items := map[string][]byte{}
	for _, record := range records {
		items[fmt.Sprintf("%s/%03d", record.Need, record.Order)] = encoded(record)
	}
	return items
}

func encoded(value any) []byte {
	content, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return content
}

func normalizeJSON(content []byte) []byte {
	var value any
	if err := json.Unmarshal(content, &value); err != nil {
		return content
	}
	return encoded(value)
}

func sortedBytesKeys(items map[string][]byte) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortRemovals(refs []RecordRef) {
	order := map[string]int{"route": 0, "node": 1, "need": 2, "item": 3, "schema": 4}
	sort.SliceStable(refs, func(left int, right int) bool {
		leftOrder := order[refs[left].Kind]
		rightOrder := order[refs[right].Kind]
		if leftOrder != rightOrder {
			return leftOrder < rightOrder
		}
		return refs[left].Key < refs[right].Key
	})
}
