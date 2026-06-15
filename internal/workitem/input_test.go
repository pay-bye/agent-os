package workitem

import (
	"errors"
	"testing"

	"github.com/pay-bye/agent-os/internal/registry"
)

func validInput(changes ...func(*SubmissionInput)) SubmissionInput {
	input := SubmissionInput{
		ID:      ID("x08"),
		Kind:    registry.ItemKindKey("x67"),
		Payload: []byte(`{"value":"x48"}`),
		Needs: []DeclaredNeedInput{{
			Kind:    registry.NeedKindKey("x12"),
			Payload: []byte(`{"need":"x48"}`),
		}},
	}
	for _, change := range changes {
		change(&input)
	}
	return input
}

type invalidSubmissionCase struct {
	name  string
	input SubmissionInput
	want  error
}

func assertInvalidSubmission(t *testing.T, test invalidSubmissionCase) {
	t.Helper()

	t.Run(test.name, func(t *testing.T) {
		_, err := NewSubmission(test.input)

		if !errors.Is(err, test.want) {
			t.Fatalf("error = %v, want %v", err, test.want)
		}
	})
}
