//go:build integration

package http_test

import (
	"context"
	"testing"
)

func TestProbesReturnBoundedResponses(t *testing.T) {
	ctx := context.Background()
	routes := handlerWithIDs(t, ctx)

	health := get(t, routes, "/health")
	readyz := get(t, routes, "/readyz")

	requireCode(t, health, 200)
	requireJSONContent(t, health)
	requireBody(t, health, map[string]any{"result": "live"})
	requireCode(t, readyz, 200)
	requireJSONContent(t, readyz)
	requireBody(t, readyz, map[string]any{
		"result": "ready",
		"checks": map[string]any{
			"startup":     "ready",
			"storage":     "ready",
			"migrations":  "ready",
			"verifier":    "ready",
			"declaration": "ready",
			"handler":     "ready",
		},
	})
}
