//go:build integration

package http_test

import (
	"context"
	"testing"
)

func TestClaimReturnsLeasedPayload(t *testing.T) {
	ctx := context.Background()
	routes := handlerWithIDs(t, ctx, "x25", "x27", "x30", "x32", "x31")
	submitItem(t, routes)

	response := claimItem(t, routes)
	token := claimToken(t, response)

	requireCode(t, response, 200)
	requireJSONContent(t, response)
	requireBody(t, response, map[string]any{
		"empty":        false,
		"lease_id":     "x16",
		"lease_token":  token,
		"work_item_id": "x08",
		"payload":      map[string]any{"value": "x75"},
		"expires_at":   "2026-05-18T12:10:00Z",
	})
}
