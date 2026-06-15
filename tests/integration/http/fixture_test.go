//go:build integration

package http_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/processlog"
	"github.com/pay-bye/agent-os/internal/storage/postgres"
	transport "github.com/pay-bye/agent-os/internal/transport/http"
	"github.com/pay-bye/agent-os/internal/transport/http/probes"
	"github.com/pay-bye/agent-os/internal/transport/http/security/credential"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

const bearerCredential = "c4p9w9hIh1Ox7r9xCh2F8nRuZx4MWi-oQ6Jq3U-mJLU"

func handlerWithIDs(t *testing.T, ctx context.Context, ids ...string) nethttp.Handler {
	t.Helper()

	db, schema := migratedSchema(t, ctx)
	insertVocabulary(t, ctx, db, schema)
	return newHandler(t, commandsFor(db, schema, ids...))
}

func handlerWithRecorder(
	t *testing.T,
	ctx context.Context,
	recorder processlog.Recorder,
	ids ...string,
) nethttp.Handler {
	t.Helper()

	db, schema := migratedSchema(t, ctx)
	insertVocabulary(t, ctx, db, schema)
	return newHandlerWithRecorder(t, commandsFor(db, schema, ids...), recorder)
}

func post(t *testing.T, routes nethttp.Handler, path string, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest("POST", path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+bearerCredential)
	response := httptest.NewRecorder()
	routes.ServeHTTP(response, request)
	return response
}

func get(t *testing.T, routes nethttp.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest("GET", path, nil)
	request.Header.Set("Authorization", "Bearer "+bearerCredential)
	response := httptest.NewRecorder()
	routes.ServeHTTP(response, request)
	return response
}

func submitItem(t *testing.T, routes nethttp.Handler) *httptest.ResponseRecorder {
	t.Helper()

	return post(t, routes, "/submit", `{
		"work_item_id": "x08",
		"item_kind": "x08",
		"payload": {"value": "x75"},
		"declared_needs": [
			{"need_kind": "x12", "payload": {"value": "x76"}}
		]
	}`)
}

func claimItem(t *testing.T, routes nethttp.Handler) *httptest.ResponseRecorder {
	t.Helper()

	return post(t, routes, "/claim", `{
		"channel_key": "x15",
		"lease_id": "x16",
		"lease_seconds": 600
	}`)
}

func ackItem(t *testing.T, routes nethttp.Handler, token string) *httptest.ResponseRecorder {
	t.Helper()

	return post(t, routes, "/ack", jsonBody(t, map[string]any{
		"lease_id":       "x16",
		"lease_token":    token,
		"declared_needs": []any{},
	}))
}

func nackItem(t *testing.T, routes nethttp.Handler, token string) *httptest.ResponseRecorder {
	t.Helper()

	return post(t, routes, "/nack", jsonBody(t, map[string]any{
		"lease_id":        "x16",
		"lease_token":     token,
		"declared_needs":  []any{},
		"failure_payload": map[string]any{"value": "x92"},
	}))
}

func extendLease(t *testing.T, routes nethttp.Handler, token string) *httptest.ResponseRecorder {
	t.Helper()

	return post(t, routes, "/extend", jsonBody(t, map[string]any{
		"lease_id":             "x16",
		"lease_token":          token,
		"requested_expires_at": "2026-05-18T12:20:00Z",
	}))
}

func heartbeatLease(t *testing.T, routes nethttp.Handler, token string) *httptest.ResponseRecorder {
	t.Helper()

	return post(t, routes, "/heartbeat", jsonBody(t, map[string]any{
		"lease_id":    "x16",
		"lease_token": token,
	}))
}

func newHandler(t *testing.T, commands kernel.Commands) nethttp.Handler {
	t.Helper()

	return newHandlerWithRecorder(t, commands, nil)
}

func newHandlerWithRecorder(
	t *testing.T,
	commands kernel.Commands,
	recorder processlog.Recorder,
) nethttp.Handler {
	t.Helper()

	return newHandlerWithOptions(t, commands, transport.WithRecorder(recorder))
}

func newHandlerWithMetrics(
	t *testing.T,
	commands kernel.Commands,
	collector *metrics.Collector,
) nethttp.Handler {
	t.Helper()

	return newHandlerWithOptions(t, commands, transport.WithMetrics(collector))
}

func newHandlerWithOptions(
	t *testing.T,
	commands kernel.Commands,
	options ...transport.Option,
) nethttp.Handler {
	t.Helper()

	verifier, err := credential.NewVerifier(verifierDigest(bearerCredential))
	if err != nil {
		t.Fatal(err)
	}
	options = append(options, transport.WithReadiness(func(context.Context) probes.Readiness {
		return probes.AllReady()
	}))
	handler, err := transport.New(commands, verifier, options...)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func verifierDigest(credential string) string {
	sum := sha256.Sum256([]byte(credential))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func migratedSchema(t *testing.T, ctx context.Context) (*sql.DB, string) {
	t.Helper()

	db := postgresfixture.Open(t)
	schema := postgresfixture.CreateSchema(t, ctx, db, "x79")
	tx := postgresfixture.Begin(t, ctx, db)
	postgresfixture.SetSearchPath(t, ctx, tx, schema)
	postgresfixture.ApplyMigrations(t, ctx, tx)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return db, schema
}

func commandsFor(db *sql.DB, schema string, ids ...string) kernel.Commands {
	return kernel.NewCommands(
		postgres.NewKernel(db, postgres.WithSearchPath(schema)),
		fixedClock{now: instant(0)},
		&sequenceIDs{values: ids},
	)
}

func insertVocabulary(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
	t.Helper()

	_, err := db.ExecContext(ctx, `
SET search_path TO `+schema+`;
INSERT INTO schema_documents (key, document) VALUES ('x01', '{"title":"x91"}');
INSERT INTO item_kinds (key, schema_key, description) VALUES ('x08', 'x01', 'x91');
INSERT INTO need_kinds (key, schema_key, description) VALUES ('x12', 'x01', 'x91');
INSERT INTO nodes (key, description) VALUES ('x17', 'x91');
INSERT INTO channels (key, node_key, description) VALUES ('x15', 'x17', 'x91');
INSERT INTO node_capabilities (node_key, need_kind_key) VALUES ('x17', 'x12');
INSERT INTO routing_rules (need_kind_key, node_key, rule_order) VALUES ('x12', 'x17', 1);
`)
	if err != nil {
		t.Fatal(err)
	}
}

func requireCode(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()

	if response.Code != want {
		t.Fatalf("code = %d, want %d, body=%s", response.Code, want, response.Body.String())
	}
}

func requireJSONContent(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q, want application/json", got)
	}
}

func requireBody(t *testing.T, response *httptest.ResponseRecorder, want map[string]any) {
	t.Helper()

	got := responseBody(t, response)
	if len(got) != len(want) {
		t.Fatalf("body = %v, want %v", got, want)
	}
	for key, wantValue := range want {
		if !equal(got[key], wantValue) {
			t.Fatalf("body[%s] = %v, want %v; body=%v", key, got[key], wantValue, got)
		}
	}
}

func claimToken(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()

	value, ok := responseBody(t, response)["lease_token"].(string)
	if !ok || strings.TrimSpace(value) == "" {
		t.Fatalf("claim token missing from body: %s", response.Body.String())
	}
	return value
}

func responseBody(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var got map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	return got
}

func jsonBody(t *testing.T, value map[string]any) string {
	t.Helper()

	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func equal(left any, right any) bool {
	leftBytes, leftErr := json.Marshal(left)
	rightBytes, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftBytes) == string(rightBytes)
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceIDs struct {
	values []string
}

func (s *sequenceIDs) Next() string {
	if len(s.values) == 0 {
		return "x35"
	}
	value := s.values[0]
	s.values = s.values[1:]
	return value
}

func instant(minute int) time.Time {
	return time.Date(2026, 5, 18, 12, minute, 0, 0, time.UTC)
}
