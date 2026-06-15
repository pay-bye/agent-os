package registry

import (
	"errors"
	"testing"
)

func TestNotFoundErrorsNameMissingVocabulary(t *testing.T) {
	tests := []notFoundCase{
		{
			name: "schema document",
			err:  SchemaDocumentNotFound(SchemaKey("x01")),
			want: `registry schema document not found: key="x01"`,
		},
		{
			name: "item kind",
			err:  ItemKindNotFound(ItemKindKey("x08")),
			want: `registry item kind not found: key="x08"`,
		},
		{
			name: "need kind",
			err:  NeedKindNotFound(NeedKindKey("x12")),
			want: `registry need kind not found: key="x12"`,
		},
	}

	for _, test := range tests {
		assertNotFoundError(t, test)
	}
}

func TestNodeVocabularyNotFoundErrors(t *testing.T) {
	tests := []notFoundCase{
		{
			name: "node",
			err:  NodeNotFound(NodeKey("x60")),
			want: `registry node not found: key="x60"`,
		},
		{
			name: "channel",
			err:  ChannelNotFound(ChannelKey("x64")),
			want: `registry channel not found: key="x64"`,
		},
		{
			name: "journal event kind",
			err:  JournalEventKindNotFound(JournalEventKindKey("x66")),
			want: `registry journal event kind not found: key="x66"`,
		},
	}

	for _, test := range tests {
		assertNotFoundError(t, test)
	}
}

func TestIsNotFoundRejectsUnrelatedErrors(t *testing.T) {
	if IsNotFound(errors.New("other failure")) {
		t.Fatal("unrelated error must not classify as not-found")
	}
}

type notFoundCase struct {
	name string
	err  error
	want string
}

func assertNotFoundError(t *testing.T, test notFoundCase) {
	t.Helper()

	t.Run(test.name, func(t *testing.T) {
		if !IsNotFound(test.err) {
			t.Fatalf("expected not-found classification, got %v", test.err)
		}
		if test.err.Error() != test.want {
			t.Fatalf("message = %q, want %q", test.err.Error(), test.want)
		}
	})
}
