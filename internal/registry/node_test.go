package registry

import (
	"errors"
	"testing"
)

func TestNewNodeRejectsInvalidInput(t *testing.T) {
	for _, test := range []invalidNodeCase{
		{
			name: "empty key",
			input: validNodeInput(func(input *NodeInput) {
				input.Key = ""
			}),
			want: ErrEmptyNodeKey,
		},
		{
			name: "empty description",
			input: validNodeInput(func(input *NodeInput) {
				input.Description = ""
			}),
			want: ErrEmptyNodeDescription,
		},
		{
			name: "empty channel",
			input: validNodeInput(func(input *NodeInput) {
				input.Channel = ""
			}),
			want: ErrEmptyChannelKey,
		},
		{
			name: "empty capabilities",
			input: validNodeInput(func(input *NodeInput) {
				input.Capabilities = nil
			}),
			want: ErrEmptyCapabilities,
		},
		{
			name: "empty capability",
			input: validNodeInput(func(input *NodeInput) {
				input.Capabilities = []NeedKindKey{""}
			}),
			want: ErrEmptyCapabilityKind,
		},
	} {
		assertInvalidNode(t, test)
	}
}

func TestNewNodePreservesCapabilities(t *testing.T) {
	input := validNodeInput()
	node, err := NewNode(input)
	if err != nil {
		t.Fatal(err)
	}

	input.Capabilities[0] = NeedKindKey("x74")
	got := node.Capabilities()
	got[0] = NeedKindKey("x74")

	if node.Key() != NodeKey("x17") {
		t.Fatalf("node key = %q, want x17", node.Key())
	}
	if node.Channel() != ChannelKey("x15") {
		t.Fatalf("channel key = %q, want x15", node.Channel())
	}
	if node.Description() != "First" {
		t.Fatalf("description = %q, want First", node.Description())
	}
	if node.Capabilities()[0] != NeedKindKey("x12") {
		t.Fatalf("capability = %q, want x12", node.Capabilities()[0])
	}
}

func TestNodeMatchesCapability(t *testing.T) {
	node, err := NewNode(validNodeInput(func(input *NodeInput) {
		input.Capabilities = []NeedKindKey{"x12", "x13"}
	}))
	if err != nil {
		t.Fatal(err)
	}

	if !node.HasCapability("x13") {
		t.Fatal("expected node to have x13")
	}
	if node.HasCapability("x14") {
		t.Fatal("unexpected x14 capability")
	}
}

func validNodeInput(changes ...func(*NodeInput)) NodeInput {
	input := NodeInput{
		Key:          NodeKey("x17"),
		Description:  "First",
		Channel:      ChannelKey("x15"),
		Capabilities: []NeedKindKey{NeedKindKey("x12")},
	}
	for _, change := range changes {
		change(&input)
	}
	return input
}

type invalidNodeCase struct {
	name  string
	input NodeInput
	want  error
}

func assertInvalidNode(t *testing.T, test invalidNodeCase) {
	t.Helper()

	t.Run(test.name, func(t *testing.T) {
		_, err := NewNode(test.input)

		if !errors.Is(err, test.want) {
			t.Fatalf("error = %v, want %v", err, test.want)
		}
	})
}
