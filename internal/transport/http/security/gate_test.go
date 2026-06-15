package security

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/transport/http/security/credential"
)

func TestOperatorKeyGateRequiresBoundaryCredentialFirst(t *testing.T) {
	called := false
	handler := RequireCredential(
		mustVerifier(t, validCredential),
		RequireOperatorKey(mustVerifier(t, otherCredential), nethttp.HandlerFunc(func(nethttp.ResponseWriter, *nethttp.Request) {
			called = true
		}), metrics.New()),
		nil,
		metrics.New(),
	)
	request := httptest.NewRequest(nethttp.MethodPost, "/private", nil)
	request.Header.Set("Operator-Key", otherCredential)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	requireCode(t, response, nethttp.StatusUnauthorized)
	requireUnauthorizedBody(t, response)
	if called {
		t.Fatal("operator handler called without boundary credential")
	}
}

func TestOperatorKeyGateRejectsMissingAndWrongKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{name: "missing"},
		{name: "wrong", key: validCredential},
		{name: "malformed", key: "two words"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			handler := RequireOperatorKey(mustVerifier(t, otherCredential), nethttp.HandlerFunc(func(nethttp.ResponseWriter, *nethttp.Request) {
				called = true
			}), metrics.New())
			request := httptest.NewRequest(nethttp.MethodPost, "/private", strings.NewReader(`{`))
			if test.key != "" {
				request.Header.Set("Operator-Key", test.key)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			requireCode(t, response, nethttp.StatusUnauthorized)
			requireUnauthorizedBody(t, response)
			if called {
				t.Fatal("operator handler called")
			}
		})
	}
}

func TestOperatorKeyGateAcceptsValidKey(t *testing.T) {
	handler := RequireOperatorKey(mustVerifier(t, otherCredential), nethttp.HandlerFunc(func(response nethttp.ResponseWriter, request *nethttp.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != `{"sample":true}` {
			t.Fatalf("body = %q", body)
		}
		response.WriteHeader(nethttp.StatusNoContent)
	}), metrics.New())
	request := httptest.NewRequest(nethttp.MethodPost, "/private", strings.NewReader(`{"sample":true}`))
	request.Header.Set("Operator-Key", otherCredential)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	requireCode(t, response, nethttp.StatusNoContent)
}

const (
	validCredential = "w8JQSjz8qsBqq4lUmX70-k3MsG6YHj5p_EyV_dLPyVk"
	otherCredential = "Po4o08mUIlP5_8pO0HbLR8rloXaHvd7s2u6H9d5aI1A"
)

func mustVerifier(t *testing.T, key string) credential.Verifier {
	t.Helper()

	verifier, err := credential.NewVerifier(verifierDigest(key))
	if err != nil {
		t.Fatal(err)
	}
	return verifier
}

func verifierDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func requireCode(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()

	if response.Code != want {
		t.Fatalf("code = %d, want %d, body=%s", response.Code, want, response.Body.String())
	}
}

func requireUnauthorizedBody(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	if response.Body.String() != "Unauthorized\n" {
		t.Fatalf("body = %q, want generic unauthorized", response.Body.String())
	}
}
