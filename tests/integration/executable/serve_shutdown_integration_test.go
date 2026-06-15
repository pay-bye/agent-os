//go:build integration

package executable_test

import (
	"context"
	nethttp "net/http"
	"os"
	"syscall"
	"testing"

	"github.com/pay-bye/agent-os/internal/transport/http/security/credential"
	"github.com/pay-bye/agent-os/tests/integration/fixtures/postgres"
)

func TestExecutableServeStopsOnSignal(t *testing.T) {
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

	url := "http://" + address + "/compatibility"
	requireStatus(t, url, statusExpectation{credential: credential.Credential, code: nethttp.StatusOK})
	if err := command.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	requireExit(t, done)
	requireStopped(t, url)
	requireLogOperations(
		t,
		command,
		"config.validate",
		"storage.migrate",
		"process.start",
		"http.accept",
		"http.complete",
		"process.stop",
	)
}
