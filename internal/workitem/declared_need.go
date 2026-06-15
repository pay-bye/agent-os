package workitem

import (
	"encoding/json"
	"errors"
	"github.com/pay-bye/agent-os/internal/registry"
)

var (
	ErrEmptyNeedKind        = errors.New("declared need kind is empty")
	ErrMalformedNeedPayload = errors.New("declared need payload is malformed JSON")
)

type DeclaredNeedInput struct {
	Kind    registry.NeedKindKey
	Target  registry.NodeKey
	Payload []byte
}

type DeclaredNeed struct {
	kind    registry.NeedKindKey
	target  registry.NodeKey
	payload []byte
}

func NewDeclaredNeed(input DeclaredNeedInput) (DeclaredNeed, error) {
	if input.Kind.String() == "" {
		return DeclaredNeed{}, ErrEmptyNeedKind
	}
	if err := validateDeclaredNeedPayload(input.Payload); err != nil {
		return DeclaredNeed{}, err
	}
	return DeclaredNeed{
		kind:    input.Kind,
		target:  input.Target,
		payload: copyDeclaredNeedPayload(input.Payload),
	}, nil
}

func (n DeclaredNeed) Kind() registry.NeedKindKey {
	return n.kind
}

func (n DeclaredNeed) Target() registry.NodeKey {
	return n.target
}

func (n DeclaredNeed) Payload() []byte {
	return copyDeclaredNeedPayload(n.payload)
}

func NewDeclaredNeeds(inputs []DeclaredNeedInput) ([]DeclaredNeed, error) {
	needs := make([]DeclaredNeed, 0, len(inputs))
	for _, input := range inputs {
		need, err := NewDeclaredNeed(input)
		if err != nil {
			return nil, err
		}
		needs = append(needs, need)
	}
	return needs, nil
}

func validateDeclaredNeedPayload(payload []byte) error {
	if len(payload) == 0 {
		return nil
	}
	if !json.Valid(payload) {
		return ErrMalformedNeedPayload
	}
	return nil
}

func copyDeclaredNeedPayload(payload []byte) []byte {
	if len(payload) == 0 {
		return nil
	}
	return append([]byte(nil), payload...)
}
