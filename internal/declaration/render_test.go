package declaration

import (
	"strings"
	"testing"
)

func TestRenderWritesIndentedDeltaWithTrailingNewline(t *testing.T) {
	body, err := Render(Delta{
		Installable: true,
		Clearances:  []RecordRef{{Kind: "routing_exclusion", Key: "x17"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	text := string(body)
	if !strings.HasSuffix(text, "\n") {
		t.Fatalf("rendered delta missing trailing newline: %q", text)
	}
	if !strings.Contains(text, `"routing_exclusion_clearances": [`) {
		t.Fatalf("rendered delta = %s", text)
	}
}
