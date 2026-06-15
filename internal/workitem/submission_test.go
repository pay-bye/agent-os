package workitem

import "testing"

func TestNewSubmissionRejectsInvalidSubmissionInput(t *testing.T) {
	for _, test := range []invalidSubmissionCase{
		{
			name: "empty identity",
			input: validInput(func(input *SubmissionInput) {
				input.ID = ""
			}),
			want: ErrEmptyID,
		},
		{
			name: "empty item kind",
			input: validInput(func(input *SubmissionInput) {
				input.Kind = ""
			}),
			want: ErrEmptyKind,
		},
		{
			name: "empty payload",
			input: validInput(func(input *SubmissionInput) {
				input.Payload = nil
			}),
			want: ErrEmptyPayload,
		},
		{
			name: "malformed payload",
			input: validInput(func(input *SubmissionInput) {
				input.Payload = []byte(`{"broken"`)
			}),
			want: ErrMalformedPayload,
		},
	} {
		assertInvalidSubmission(t, test)
	}
}

func TestNewSubmissionPreservesOpaquePayload(t *testing.T) {
	input := validInput()
	submission, err := NewSubmission(input)
	if err != nil {
		t.Fatal(err)
	}

	input.Payload[0] = '['
	got := submission.Payload()
	got[0] = '['

	if string(submission.Payload()) != `{"value":"x48"}` {
		t.Fatalf("submission payload must not expose mutable alias")
	}
}
