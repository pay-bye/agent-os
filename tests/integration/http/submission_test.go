//go:build integration

package http_test

import (
	"context"
	"testing"
)

func TestSubmitReturnsRouteResult(t *testing.T) {
	ctx := context.Background()
	routes := handlerWithIDs(t, ctx, "x25", "x27", "x30", "x32")

	response := submitItem(t, routes)

	requireCode(t, response, 200)
	requireJSONContent(t, response)
	requireBody(t, response, map[string]any{
		"work_item_id": "x08",
		"routed":       true,
		"channel_key":  "x15",
	})
}
