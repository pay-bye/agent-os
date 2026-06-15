//go:build integration

package executable_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	nethttp "net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/pay-bye/agent-os/internal/declaration"
)

type serveInput struct {
	schema         string
	declaration    string
	address        string
	verifierDigest string
}

type statusExpectation struct {
	credential string
	code       int
}

type deltaInput struct {
	databaseURL string
	declaration string
}

type processRecord struct {
	Operation string `json:"operation"`
}

func runDeltaCommand(
	t *testing.T,
	binary string,
	verb string,
	input deltaInput,
) declaration.Delta {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command := exec.Command(
		binary,
		verb,
		"--database-url", input.databaseURL,
		"--from", input.declaration,
	)
	command.Dir = findRoot(t)
	command.Env = os.Environ()
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("%s failed: %v\n%s", verb, err, redactedOutput(command))
	}
	var delta declaration.Delta
	if err := json.Unmarshal(stdout.Bytes(), &delta); err != nil {
		t.Fatal(err)
	}
	return delta
}

func runFailingDeltaCommand(
	t *testing.T,
	binary string,
	verb string,
	input deltaInput,
) {
	t.Helper()

	var output bytes.Buffer
	command := exec.Command(
		binary,
		verb,
		"--database-url", input.databaseURL,
		"--from", input.declaration,
	)
	command.Dir = findRoot(t)
	command.Env = os.Environ()
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err == nil {
		t.Fatalf("%s succeeded unexpectedly: %s", verb, output.String())
	}
}

func serveCommand(t *testing.T, input serveInput) *exec.Cmd {
	t.Helper()

	binary := buildBinary(t)
	command := exec.Command(
		binary,
		"serve",
		"--database-url",
		withSearchPath(t, os.Getenv("DATABASE_URL"), input.schema),
		"--listen",
		input.address,
		"--from",
		input.declaration,
		"--verifier-digest",
		input.verifierDigest,
		"--shutdown-grace",
		"1s",
	)
	command.Dir = findRoot(t)
	command.Env = os.Environ()
	command.Stdout = &bytes.Buffer{}
	command.Stderr = &bytes.Buffer{}
	return command
}

func buildBinary(t *testing.T) string {
	t.Helper()

	binary := filepath.Join(t.TempDir(), "substrate")
	command := exec.Command("go", "build", "-o", binary, "./cmd/substrate")
	command.Dir = findRoot(t)
	command.Env = os.Environ()
	command.Stdout = &bytes.Buffer{}
	command.Stderr = &bytes.Buffer{}
	if err := command.Run(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, redactedOutput(command))
	}
	return binary
}

func waitForProcess(command *exec.Cmd) <-chan error {
	done := make(chan error, 1)
	go func() {
		done <- command.Wait()
	}()
	return done
}

func stopProcess(command *exec.Cmd, done <-chan error) {
	if command.Process == nil {
		return
	}
	_ = command.Process.Signal(syscall.SIGTERM)
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = command.Process.Kill()
	}
}

func requireExit(t *testing.T, done <-chan error) {
	t.Helper()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("serve did not stop within grace")
	}
}

func requireStatus(t *testing.T, url string, expected statusExpectation) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		status, err := requestStatus(url, expected.credential)
		if err == nil && status == expected.code {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("status = %d, error = %v, want %d", status, err, expected.code)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func requestStatus(url string, credential string) (int, error) {
	request, err := nethttp.NewRequest(nethttp.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	if credential != "" {
		request.Header.Set("Authorization", "Bearer "+credential)
	}
	response, err := nethttp.DefaultClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	return response.StatusCode, nil
}

func requireJSON(
	t *testing.T,
	url string,
	expected statusExpectation,
	want map[string]any,
) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		code, body, err := requestJSON(url, expected.credential)
		if err == nil && code == expected.code && matches(body, want) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("response = %d %v, error = %v, want %d %v", code, body, err, expected.code, want)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func requestJSON(url string, credential string) (int, map[string]any, error) {
	request, err := nethttp.NewRequest(nethttp.MethodGet, url, nil)
	if err != nil {
		return 0, nil, err
	}
	if credential != "" {
		request.Header.Set("Authorization", "Bearer "+credential)
	}
	response, err := nethttp.DefaultClient.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return response.StatusCode, nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(content, &body); err != nil {
		return response.StatusCode, nil, err
	}
	return response.StatusCode, body, nil
}

func matches(got map[string]any, want map[string]any) bool {
	gotBytes, gotErr := json.Marshal(got)
	wantBytes, wantErr := json.Marshal(want)
	return gotErr == nil && wantErr == nil && string(gotBytes) == string(wantBytes)
}

func requireStopped(t *testing.T, url string) {
	t.Helper()

	_, err := requestStatus(url, "")
	if err == nil {
		t.Fatal("serve accepted a request after shutdown")
	}
}

func requireContains(t *testing.T, text string, want string) {
	t.Helper()

	if !strings.Contains(text, want) {
		t.Fatalf("expected %q to contain %q", text, want)
	}
}

func freeAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().String()
}

func findRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if exists(filepath.Join(dir, "go.mod")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("source root not found")
		}
		dir = parent
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !errors.Is(err, os.ErrNotExist)
}

func redactedOutput(command *exec.Cmd) string {
	var parts []string
	if out, ok := command.Stdout.(*bytes.Buffer); ok {
		parts = append(parts, out.String())
	}
	if out, ok := command.Stderr.(*bytes.Buffer); ok {
		parts = append(parts, out.String())
	}
	return strings.Join(parts, "\n")
}

func requireLogOperations(t *testing.T, command *exec.Cmd, operations ...string) {
	t.Helper()

	records := processRecords(t, command)
	if len(records) != len(operations) {
		t.Fatalf("records = %+v, want operations %v", records, operations)
	}
	for index, operation := range operations {
		if records[index].Operation != operation {
			t.Fatalf("record %d operation = %q, want %q", index, records[index].Operation, operation)
		}
	}
}

func processRecords(t *testing.T, command *exec.Cmd) []processRecord {
	t.Helper()

	stderr, ok := command.Stderr.(*bytes.Buffer)
	if !ok {
		t.Fatal("stderr buffer missing")
	}
	var records []processRecord
	for _, line := range strings.Split(stderr.String(), "\n") {
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var record processRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return records
}
