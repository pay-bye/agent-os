package credential

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

type verifierFile struct {
	Digests []string `json:"digests"`
}

// ReadVerifierFile reads an operator-chosen JSON verifier file.
func ReadVerifierFile(path string) (Verifier, error) {
	digests, err := ReadVerifierDigests(path)
	if err != nil {
		return Verifier{}, err
	}
	return NewVerifier(digests...)
}

func ReadVerifierDigests(path string) ([]string, error) {
	root, name, err := openRoot(path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	info, err := root.Stat(name)
	if err != nil {
		return nil, err
	}
	if info.IsDir() || info.Mode().Perm() != 0o600 {
		return nil, errInvalidVerifier
	}
	content, err := root.ReadFile(name)
	if err != nil {
		return nil, err
	}
	var file verifierFile
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return nil, errInvalidVerifier
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, errInvalidVerifier
	}
	if _, err := NewVerifier(file.Digests...); err != nil {
		return nil, err
	}
	return append([]string(nil), file.Digests...), nil
}

func openRoot(path string) (*os.Root, string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, "", err
	}
	root, err := os.OpenRoot(filepath.Dir(absolute))
	if err != nil {
		return nil, "", err
	}
	return root, filepath.Base(absolute), nil
}
