//go:build integration

package executable_test

import (
	"context"
	nethttp "net/http"
	"os"
	"testing"

	"github.com/pay-bye/agent-os/internal/transport/http/security/credential"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

func TestExecutableServePublishesCredentialedProbes(t *testing.T) {
	ctx := context.Background()
	db := postgresfixture.Open(t)
	schema := postgresfixture.CreateSchema(t, ctx, db, "x96")
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
	requireStatus(t, root+"/health", statusExpectation{code: nethttp.StatusUnauthorized})
	requireJSON(t, root+"/health", statusExpectation{credential: credential.Credential, code: nethttp.StatusOK}, map[string]any{
		"result": "live",
	})
	requireJSON(t, root+"/readyz", statusExpectation{credential: credential.Credential, code: nethttp.StatusOK}, map[string]any{
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
