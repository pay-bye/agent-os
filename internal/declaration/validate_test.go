package declaration

import (
	"strings"
	"testing"
)

func TestParseRejectsOpaqueInvalidStructure(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "duplicate key", body: "version: 1\nversion: 1\n", want: "duplicate_key"},
		{
			name: "unknown field",
			body: strings.Replace(validDocument(), "routes:", "unknown_member: {}\nroutes:", 1),
			want: "unknown_field",
		},
		{
			name: "unknown item field",
			body: strings.Replace(validDocument(), "description: x21", "description: x21\n    extra: value", 1),
			want: "unknown_field",
		},
		{
			name: "missing reference",
			body: strings.Replace(validDocument(), "schema: x01", "schema: x09", 1),
			want: "unknown_reference",
		},
		{
			name: "route outside capability",
			body: strings.Replace(validDocument(), "accepts:\n      - x12", "accepts: []", 1),
			want: "route_capability",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse([]byte(test.body))

			requireError(t, err, test.want)
		})
	}
}
