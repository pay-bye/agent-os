package workitem

import (
	"testing"

	"github.com/pay-bye/agent-os/internal/registry"
)

func TestNewSubmissionRejectsInvalidDeclaredNeedInput(t *testing.T) {
	for _, test := range []invalidSubmissionCase{
		{
			name: "empty need kind",
			input: validInput(func(input *SubmissionInput) {
				input.Needs = []DeclaredNeedInput{{Kind: ""}}
			}),
			want: ErrEmptyNeedKind,
		},
		{
			name: "malformed need payload",
			input: validInput(func(input *SubmissionInput) {
				input.Needs = []DeclaredNeedInput{{
					Kind:    registry.NeedKindKey("x12"),
					Payload: []byte(`{"broken"`),
				}}
			}),
			want: ErrMalformedNeedPayload,
		},
	} {
		assertInvalidSubmission(t, test)
	}
}

func TestNewDeclaredNeedRejectsInvalidInput(t *testing.T) {
	for _, test := range []struct {
		name  string
		input DeclaredNeedInput
		want  error
	}{
		{
			name:  "empty need kind",
			input: DeclaredNeedInput{Kind: ""},
			want:  ErrEmptyNeedKind,
		},
		{
			name: "malformed need payload",
			input: DeclaredNeedInput{
				Kind:    registry.NeedKindKey("x12"),
				Payload: []byte(`{"broken"`),
			},
			want: ErrMalformedNeedPayload,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewDeclaredNeed(test.input)
			if err != test.want {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestNewDeclaredNeedPreservesPayload(t *testing.T) {
	input := DeclaredNeedInput{
		Kind:    registry.NeedKindKey("x12"),
		Payload: []byte(`{"need":"x48"}`),
		Target:  registry.NodeKey("x17"),
	}

	need, err := NewDeclaredNeed(input)
	if err != nil {
		t.Fatal(err)
	}

	input.Payload[0] = '['
	payload := need.Payload()
	payload[0] = '['

	if need.Kind() != registry.NeedKindKey("x12") {
		t.Fatalf("need kind = %q, want x12", need.Kind())
	}
	if need.Target() != registry.NodeKey("x17") {
		t.Fatalf("need target = %q, want x17", need.Target())
	}
	if string(need.Payload()) != `{"need":"x48"}` {
		t.Fatalf("declared need payload must not expose mutable alias")
	}
}

func TestNewDeclaredNeedAcceptsEmptyPayload(t *testing.T) {
	need, err := NewDeclaredNeed(DeclaredNeedInput{Kind: registry.NeedKindKey("x12")})
	if err != nil {
		t.Fatal(err)
	}

	if len(need.Payload()) != 0 {
		t.Fatalf("declared need payload length = %d, want 0", len(need.Payload()))
	}
}

func TestNewDeclaredNeedsReturnsValidatedNeeds(t *testing.T) {
	needs, err := NewDeclaredNeeds([]DeclaredNeedInput{
		{Kind: registry.NeedKindKey("x12"), Payload: []byte(`{"need":"x48"}`)},
		{Kind: registry.NeedKindKey("x13"), Payload: []byte(`{"need":"x49"}`)},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(needs) != 2 {
		t.Fatalf("need count = %d, want 2", len(needs))
	}
	if needs[1].Kind() != registry.NeedKindKey("x13") {
		t.Fatalf("second need kind = %q, want x13", needs[1].Kind())
	}
}

func TestNewDeclaredNeedsPropagatesValidationErrors(t *testing.T) {
	_, err := NewDeclaredNeeds([]DeclaredNeedInput{
		{Kind: registry.NeedKindKey("x12"), Payload: []byte(`{"need":"x48"}`)},
		{Kind: registry.NeedKindKey("x13"), Payload: []byte(`{"broken"`)},
	})

	if err != ErrMalformedNeedPayload {
		t.Fatalf("error = %v, want malformed need payload", err)
	}
}

func TestNewSubmissionPreservesDeclaredNeeds(t *testing.T) {
	input := validInput()
	submission, err := NewSubmission(input)
	if err != nil {
		t.Fatal(err)
	}

	input.Needs[0].Payload[0] = '['
	needs := submission.DeclaredNeeds()
	needs[0] = DeclaredNeed{}
	needPayload := submission.DeclaredNeeds()[0].Payload()
	needPayload[0] = '['

	got := submission.DeclaredNeeds()[0]
	if got.Kind() != registry.NeedKindKey("x12") {
		t.Fatalf("need kind = %q, want x12", got.Kind())
	}
	if string(got.Payload()) != `{"need":"x48"}` {
		t.Fatalf("declared need payload must not expose mutable alias")
	}
}
