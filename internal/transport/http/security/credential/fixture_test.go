package credential

import (
	"strings"
	"testing"
)

const (
	emptyDigest  = "47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU"
	filledDigest = "ii-Ly1VyqJrpaRhq8OA8Vsh8Ir6Yy4Hgx572nIAxlnA"
)

func requireError(t *testing.T, err error, text string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error containing %q", text)
	}
	if !strings.Contains(err.Error(), text) {
		t.Fatalf("error = %v, want %q", err, text)
	}
}
