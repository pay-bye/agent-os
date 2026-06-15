//go:build integration

package executable_test

import (
	"context"
	"database/sql"
	"io"
	nethttp "net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/transport/http/security/credential"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

func TestExecutableServePublishesCredentialedMetrics(t *testing.T) {
	ctx := context.Background()
	db := postgresfixture.Open(t)
	schema := postgresfixture.CreateSchema(t, ctx, db, "x97")
	declaration := writeVocabulary(t)
	credential, err := credential.GenerateCredential()
	if err != nil {
		t.Fatal(err)
	}
	binary := buildBinary(t)
	runDeltaCommand(t, binary, "apply", deltaInput{
		databaseURL: withSearchPath(t, os.Getenv("DATABASE_URL"), schema),
		declaration: declaration,
	})
	insertMetricEntry(t, ctx, db, schema)

	address := freeAddress(t)
	command := serveCommand(t, serveInput{
		schema:         schema,
		declaration:    declaration,
		address:        address,
		verifierDigest: credential.VerifierDigest,
	})
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	done := waitForProcess(command)
	t.Cleanup(func() {
		stopProcess(command, done)
	})

	root := "http://" + address
	requireStatus(t, root+"/metrics", statusExpectation{code: nethttp.StatusUnauthorized})
	requireMetrics(t, root+"/metrics", credential.Credential, []string{
		"# TYPE queue_depth gauge\n",
		`queue_depth{channel_class="all"} 1` + "\n",
		"# TYPE build_info gauge\n",
	})
}

func insertMetricEntry(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
	t.Helper()

	_, err := db.ExecContext(ctx, `
INSERT INTO `+schema+`.channel_entries (id, channel_key, work_item_id, enqueued_at, available_at)
VALUES ('x71', 'x17', 'x72', $1, $1)`,
		time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
}

func requireMetrics(t *testing.T, url string, credential string, lines []string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		code, body, err := requestText(url, credential)
		if err == nil && code == nethttp.StatusOK && containsLines(body, lines) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("metrics = %d %q, error = %v, want lines %v", code, body, err, lines)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func requestText(url string, credential string) (int, string, error) {
	request, err := nethttp.NewRequest(nethttp.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}
	if credential != "" {
		request.Header.Set("Authorization", "Bearer "+credential)
	}
	response, err := nethttp.DefaultClient.Do(request)
	if err != nil {
		return 0, "", err
	}
	defer response.Body.Close()
	content, err := io.ReadAll(response.Body)
	return response.StatusCode, string(content), err
}

func containsLines(body string, lines []string) bool {
	for _, line := range lines {
		if !strings.Contains(body, line) {
			return false
		}
	}
	return true
}
