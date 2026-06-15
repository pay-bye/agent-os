package credential

import (
	"errors"
)

type VerifierInput struct {
	Digests []string
	File    string
}

func LoadVerifier(input VerifierInput) (Verifier, error) {
	digests := append([]string(nil), input.Digests...)
	if input.File != "" {
		fileDigests, err := ReadVerifierDigests(input.File)
		if err != nil {
			return Verifier{}, err
		}
		digests = append(digests, fileDigests...)
	}
	if len(digests) == 0 {
		return Verifier{}, errors.New("missing_verifier")
	}
	return NewVerifier(digests...)
}
