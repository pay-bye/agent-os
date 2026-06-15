package channel

import (
	"errors"
	"testing"
)

func TestNewPreparationRejectsInvalidInput(t *testing.T) {
	for _, test := range []invalidPreparationCase{
		{
			name: "empty lease",
			input: validPreparationInput(func(input *PreparationInput) {
				input.Lease = ""
			}),
			want: ErrEmptyLeaseID,
		},
		{
			name: "empty kind",
			input: validPreparationInput(func(input *PreparationInput) {
				input.Kind = ""
			}),
			want: ErrEmptyPreparationKind,
		},
	} {
		assertInvalidPreparation(t, test)
	}
}

func TestNewPreparationReportsFields(t *testing.T) {
	preparation, err := NewPreparation(validPreparationInput())
	if err != nil {
		t.Fatal(err)
	}

	if preparation.Lease() != LeaseID("x16") {
		t.Fatalf("lease identity = %q, want x16", preparation.Lease())
	}
	if preparation.Kind() != Ack {
		t.Fatalf("kind = %q, want ack", preparation.Kind())
	}
}

func validPreparationInput(changes ...func(*PreparationInput)) PreparationInput {
	input := PreparationInput{
		Lease: LeaseID("x16"),
		Kind:  Ack,
	}
	for _, change := range changes {
		change(&input)
	}
	return input
}

type invalidPreparationCase struct {
	name  string
	input PreparationInput
	want  error
}

func assertInvalidPreparation(t *testing.T, test invalidPreparationCase) {
	t.Helper()

	t.Run(test.name, func(t *testing.T) {
		_, err := NewPreparation(test.input)

		if !errors.Is(err, test.want) {
			t.Fatalf("error = %v, want %v", err, test.want)
		}
	})
}
