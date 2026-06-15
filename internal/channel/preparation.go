package channel

import (
	"errors"
)

const (
	Ack  PreparationKind = "ack"
	Nack PreparationKind = "nack"
)

var ErrEmptyPreparationKind = errors.New("lease preparation kind is empty")

type PreparationKind string

type PreparationInput struct {
	Lease LeaseID
	Kind  PreparationKind
}

type Preparation struct {
	lease LeaseID
	kind  PreparationKind
}

func NewPreparation(input PreparationInput) (Preparation, error) {
	if err := validatePreparationInput(input); err != nil {
		return Preparation{}, err
	}
	return Preparation{lease: input.Lease, kind: input.Kind}, nil
}

func (p Preparation) Lease() LeaseID {
	return p.lease
}

func (p Preparation) Kind() PreparationKind {
	return p.kind
}

func validatePreparationInput(input PreparationInput) error {
	if blank(input.Lease.String()) {
		return ErrEmptyLeaseID
	}
	if blank(string(input.Kind)) {
		return ErrEmptyPreparationKind
	}
	return nil
}
