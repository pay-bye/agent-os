//go:build integration

package http_test

import (
	"context"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/internal/metrics"
)

func TestOperationReadsEmitBoundedObservation(t *testing.T) {
	ctx := context.Background()
	collector := metrics.New()
	db, schema := migratedSchema(t, ctx)
	insertVocabulary(t, ctx, db, schema)
	routes := newHandlerWithMetrics(t, commandsFor(db, schema), collector)

	response := get(t, routes, "/operations/channels")

	requireCode(t, response, 409)
	requireMetric(t, collector, `requests_total{operation="operations",result="failed",protocol="http"} 1`)
}

func requireMetric(t *testing.T, collector *metrics.Collector, line string) {
	t.Helper()

	text := collector.Text(context.Background())
	if !strings.Contains(text, line+"\n") {
		t.Fatalf("metric line %q missing from:\n%s", line, text)
	}
}
