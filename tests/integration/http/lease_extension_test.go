//go:build integration

package http_test

import (
	"context"
	"testing"
)

func TestExtendReturnsUpdatedLease(t *testing.T) {
	ctx := context.Background()
	routes := handlerWithIDs(t, ctx, "x25", "x27", "x30", "x32", "x31")
	submitItem(t, routes)
	claim := claimItem(t, routes)

	response := extendLease(t, routes, claimToken(t, claim))

	requireCode(t, response, 200)
	requireJSONContent(t, response)
	requireBody(t, response, map[string]any{
		"lease_id":   "x16",
		"expires_at": "2026-05-18T12:20:00Z",
	})
}
