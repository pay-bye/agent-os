package ids

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestNewReturnsOpaqueURLID(t *testing.T) {
	id, err := New()
	if err != nil {
		t.Fatal(err)
	}

	requireOpaqueURLID(t, id)
}

func TestRandomNextReturnsOpaqueURLID(t *testing.T) {
	id := Random{}.Next()

	requireOpaqueURLID(t, id)
}

func TestNewReportsSourceFailure(t *testing.T) {
	_, err := newFrom(failingReader{})

	if err == nil {
		t.Fatal("expected source failure")
	}
	if !errors.Is(err, errFailedRead) {
		t.Fatalf("error = %v, want %v", err, errFailedRead)
	}
}

func requireOpaqueURLID(t *testing.T, id string) {
	t.Helper()

	raw, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		t.Fatalf("id is not raw URL base64: %v", err)
	}
	if len(raw) != 16 || strings.Contains(id, "=") || base64.RawURLEncoding.EncodeToString(raw) != id {
		t.Fatalf("id = %q", id)
	}
}

var errFailedRead = errors.New("failed_read")

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errFailedRead
}
