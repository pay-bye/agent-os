package workitem

import (
	"encoding/json"
	"errors"
	"github.com/pay-bye/agent-os/internal/registry"
)

var (
	ErrEmptyID          = errors.New("submission identity is empty")
	ErrEmptyKind        = errors.New("submission kind is empty")
	ErrEmptyPayload     = errors.New("submission payload is empty")
	ErrMalformedPayload = errors.New("submission payload is malformed JSON")
)

type ID string

func (id ID) String() string {
	return string(id)
}

type SubmissionInput struct {
	ID      ID
	Kind    registry.ItemKindKey
	Payload []byte
	Needs   []DeclaredNeedInput
}

type Submission struct {
	id      ID
	kind    registry.ItemKindKey
	payload []byte
	needs   []DeclaredNeed
}

func NewSubmission(input SubmissionInput) (Submission, error) {
	if err := validateSubmissionInput(input); err != nil {
		return Submission{}, err
	}
	needs, err := NewDeclaredNeeds(input.Needs)
	if err != nil {
		return Submission{}, err
	}
	return Submission{
		id:      input.ID,
		kind:    input.Kind,
		payload: copySubmissionPayload(input.Payload),
		needs:   needs,
	}, nil
}

func (s Submission) ID() ID {
	return s.id
}

func (s Submission) Kind() registry.ItemKindKey {
	return s.kind
}

func (s Submission) Payload() []byte {
	return copySubmissionPayload(s.payload)
}

func (s Submission) DeclaredNeeds() []DeclaredNeed {
	return append([]DeclaredNeed(nil), s.needs...)
}

func validateSubmissionInput(input SubmissionInput) error {
	if input.ID.String() == "" {
		return ErrEmptyID
	}
	if input.Kind.String() == "" {
		return ErrEmptyKind
	}
	return validateSubmissionPayload(input.Payload)
}

func validateSubmissionPayload(payload []byte) error {
	if len(payload) == 0 {
		return ErrEmptyPayload
	}
	if !json.Valid(payload) {
		return ErrMalformedPayload
	}
	return nil
}

func copySubmissionPayload(payload []byte) []byte {
	if payload == nil {
		return nil
	}
	return append([]byte(nil), payload...)
}
