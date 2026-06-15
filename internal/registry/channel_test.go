package registry

import (
	"errors"
	"testing"
)

func TestNewChannelRejectsInvalidInput(t *testing.T) {
	for _, test := range []invalidChannelCase{
		{
			name: "empty key",
			input: validChannelInput(func(input *ChannelInput) {
				input.Key = ""
			}),
			want: ErrEmptyChannelKey,
		},
		{
			name: "empty node",
			input: validChannelInput(func(input *ChannelInput) {
				input.Node = ""
			}),
			want: ErrEmptyNodeKey,
		},
		{
			name: "empty description",
			input: validChannelInput(func(input *ChannelInput) {
				input.Description = ""
			}),
			want: ErrEmptyChannelDescription,
		},
	} {
		assertInvalidChannel(t, test)
	}
}

func TestNewChannelReportsFields(t *testing.T) {
	channel, err := NewChannel(validChannelInput())
	if err != nil {
		t.Fatal(err)
	}

	if channel.Key() != ChannelKey("x15") {
		t.Fatalf("channel key = %q, want x15", channel.Key())
	}
	if channel.Node() != NodeKey("x17") {
		t.Fatalf("node key = %q, want x17", channel.Node())
	}
	if channel.Description() != "First" {
		t.Fatalf("description = %q, want First", channel.Description())
	}
}

func validChannelInput(changes ...func(*ChannelInput)) ChannelInput {
	input := ChannelInput{
		Key:         ChannelKey("x15"),
		Node:        NodeKey("x17"),
		Description: "First",
	}
	for _, change := range changes {
		change(&input)
	}
	return input
}

type invalidChannelCase struct {
	name  string
	input ChannelInput
	want  error
}

func assertInvalidChannel(t *testing.T, test invalidChannelCase) {
	t.Helper()

	t.Run(test.name, func(t *testing.T) {
		_, err := NewChannel(test.input)

		if !errors.Is(err, test.want) {
			t.Fatalf("error = %v, want %v", err, test.want)
		}
	})
}
