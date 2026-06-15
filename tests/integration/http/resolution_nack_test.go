//go:build integration

package http_test

import (
	"context"
	"testing"
)

func TestNackResolvesClaimedItem(t *testing.T) {
	ctx := context.Background()
	routes := handlerWithIDs(t, ctx, "x25", "x27", "x30", "x32", "x31", "x72", "x73")
	submitItem(t, routes)
	claim := claimItem(t, routes)

	response := nackItem(t, routes, claimToken(t, claim))

	requireCode(t, response, 200)
	requireJSONContent(t, response)
	requireBody(t, response, map[string]any{"resolved": true, "routed": false})
}
